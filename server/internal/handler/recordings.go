package handler

import (
	"context"
	"errors"
	"io"
	"log"
	"path/filepath"
	"strings"

	"github.com/google/uuid"

	"github.com/ChrQR/meeting-assistant/server/internal/api"
	"github.com/ChrQR/meeting-assistant/server/internal/types"
)

func (s *Handlers) UploadRecording(ctx context.Context, req api.UploadRecordingRequestObject) (api.UploadRecordingResponseObject, error) {
	var title string
	var fileBytes []byte
	var extension string
	var fileName string
	for {
		part, err := req.Body.NextPart()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}

		switch part.FormName() {
		case "title":
			data, err := io.ReadAll(part)
			if err != nil {
				return nil, err
			}
			title = string(data)
		case "file":
			fileName := part.FileName()

			// 2. Extract extension (e.g. ".mp3" -> "mp3")
			ext := filepath.Ext(fileName)
			extension = strings.TrimPrefix(ext, ".")
			data, err := io.ReadAll(part)
			if err != nil {
				return nil, err
			}
			fileBytes = data
		}
	}
	if title == "" || len(fileBytes) == 0 {
		return api.UploadRecording400Response{}, errors.New("missing required form fields")
	}
	if err := s.TranscriptionService.ProcessRecording(ctx, types.Recording{
		Title:     title,
		FileData:  fileBytes,
		Extension: extension,
		FileName:  fileName,
		UserPath:  "christian", // development value. will be user id later, extracted by auth.
	}); err != nil {
		log.Print(err)
		return api.UploadRecording400Response{}, err
	}
	return api.UploadRecording201Response{}, nil
}

func (s *Handlers) GetRecordings(ctx context.Context, req api.GetRecordingsRequestObject) (api.GetRecordingsResponseObject, error) {
	data, err := s.StoreService.GetRecordings(ctx)
	if err != nil {
		return api.GetRecordings500Response{}, err
	}

	result := make(api.GetRecordings200JSONResponse, 0, len(data))

	for _, rec := range data {
		result = append(result, struct {
			Extension *string    `json:"extension,omitempty"`
			Filename  *string    `json:"filename,omitempty"`
			Id        *uuid.UUID `json:"id,omitempty"`
			Status    *string    `json:"status,omitempty"`
		}{
			Extension: &rec.Extension,
			Filename:  &rec.Filename,
			Status:    new(string(rec.Status)),
			Id:        &rec.ID,
		})
	}

	return result, nil
}
