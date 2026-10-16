package handler

import (
	"context"
	"errors"
	"io"

	"github.com/ChrQR/meeting-assistant/server/internal/api"
	"github.com/ChrQR/meeting-assistant/server/internal/types"
)

func (s *Handlers) UploadRecording(ctx context.Context, req api.UploadRecordingRequestObject) (api.UploadRecordingResponseObject, error) {
	var title string
	var fileBytes []byte
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
		Title:    title,
		FileData: fileBytes,
	}); err != nil {
		return api.UploadRecording400Response{}, err
	}
	return api.UploadRecording201Response{}, nil
}
