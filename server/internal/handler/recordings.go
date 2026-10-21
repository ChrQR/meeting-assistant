package handler

import (
	"bytes"
	"errors"
	"io"
	"mime"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/ChrQR/meeting-assistant/server/internal/api"
	"github.com/ChrQR/meeting-assistant/server/internal/types"
)

func (s *Handlers) UploadRecording(w http.ResponseWriter, r *http.Request) {
	userID, ok := s.requireUserID(w, r)
	if !ok {
		return
	}

	var title string
	var fileBytes []byte
	var extension string
	var fileName string
	var meetingAtValue string

	reader, err := r.MultipartReader()
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "expected multipart/form-data body"})
		return
	}

	for {
		part, err := reader.NextPart()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			writeError(w, "UploadRecording", err)
			return
		}

		switch part.FormName() {
		case "title":
			data, err := io.ReadAll(part)
			if err != nil {
				writeError(w, "UploadRecording", err)
				return
			}
			title = string(data)
		case "meeting_at":
			data, err := io.ReadAll(part)
			if err != nil {
				writeError(w, "UploadRecording", err)
				return
			}
			meetingAtValue = string(data)
		case "file":
			fileName = part.FileName()

			// 2. Extract extension (e.g. ".mp3" -> "mp3")
			ext := filepath.Ext(fileName)
			extension = strings.TrimPrefix(ext, ".")
			data, err := io.ReadAll(part)
			if err != nil {
				writeError(w, "UploadRecording", err)
				return
			}
			fileBytes = data
		}
	}
	if title == "" || len(fileBytes) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing required form fields"})
		return
	}

	meetingAt, err := types.ParseMeetingAt(meetingAtValue)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	recordingID, err := s.TranscriptionService.StartRecording(r.Context(), types.Recording{
		Title:     title,
		FileData:  fileBytes,
		Extension: extension,
		FileName:  fileName,
		UserPath:  userID.String(),
		MeetingAt: meetingAt,
		UserID:    userID,
	})
	if err != nil {
		writeError(w, "UploadRecording", err)
		return
	}

	// Processing happens in the background; the client can keep using the app.
	writeJSON(w, http.StatusAccepted, api.Recording{
		Id:        recordingID,
		Filename:  fileName,
		Extension: extension,
		Status:    string(types.RecordingPending),
	})
}

func (s *Handlers) GetRecordings(w http.ResponseWriter, r *http.Request) {
	userID, ok := s.requireUserID(w, r)
	if !ok {
		return
	}

	data, err := s.StoreService.GetRecordings(r.Context(), userID)
	if err != nil {
		writeError(w, "GetRecordings", err)
		return
	}

	result := make([]api.Recording, 0, len(data))
	for _, rec := range data {
		result = append(result, api.Recording{
			Extension: rec.Extension,
			Filename:  rec.Filename,
			Status:    string(rec.Status),
			Id:        rec.ID,
		})
	}

	writeJSON(w, http.StatusOK, result)
}

// GetRecordingAudio streams the raw recording from object storage. It uses
// http.ServeContent so that HTTP range requests are supported, enabling the
// browser to seek within the audio.
func (s *Handlers) GetRecordingAudio(w http.ResponseWriter, r *http.Request, id openapi_types.UUID) {
	userID, ok := s.requireUserID(w, r)
	if !ok {
		return
	}

	data, file, err := s.StoreService.GetRecordingAudio(r.Context(), userID, id)
	if err != nil {
		writeError(w, "GetRecordingAudio", err)
		return
	}

	contentType := mime.TypeByExtension("." + strings.TrimPrefix(file.Extension, "."))
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", `inline; filename="`+file.Filename+"."+file.Extension+`"`)
	w.Header().Set("Accept-Ranges", "bytes")
	w.Header().Set("Cache-Control", "private, max-age=3600")

	http.ServeContent(w, r, file.Filename+"."+file.Extension, time.Time{}, bytes.NewReader(data))
}
