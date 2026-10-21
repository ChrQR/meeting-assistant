package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/ChrQR/meeting-assistant/server/internal/client"
	"github.com/ChrQR/meeting-assistant/server/internal/repository"
	"github.com/ChrQR/meeting-assistant/server/internal/types"
)

const (
	// processingTimeout bounds the whole background pipeline so a stuck upstream
	// cannot leave a recording in a non-terminal state forever.
	processingTimeout = 2 * time.Hour
)

type transcriptionService struct {
	transcriptionClient client.TranscriptionClient
	storageClient       client.StorageClient
	embeddingClient     client.EmbeddingClient
	summaryClient       client.SummaryClient
	mediaClient         client.MediaClient
	dbRepository        repository.DBRepository
}

type TranscriptionService interface {
	// StartRecording stores the recording row and processes it in the background,
	// returning as soon as the row exists. Processing continues even if the
	// originating HTTP request ends.
	StartRecording(ctx context.Context, file types.Recording) (uuid.UUID, error)
	// ProcessRecording runs the full pipeline for an already-created recording.
	ProcessRecording(ctx context.Context, file types.Recording, recordingID uuid.UUID) error
}

func NewTranscriptionService(
	transcriptionClient client.TranscriptionClient,
	storageClient client.StorageClient,
	embeddingClient client.EmbeddingClient,
	summaryClient client.SummaryClient,
	mediaClient client.MediaClient,
	dbRepository repository.DBRepository,
) TranscriptionService {
	return &transcriptionService{
		transcriptionClient,
		storageClient,
		embeddingClient,
		summaryClient,
		mediaClient,
		dbRepository,
	}
}

// transcribeChunked splits long audio and transcribes each chunk, stitching the
// results back into a single transcript for the original recording. Short
// recordings are sent as a single request.
func (s *transcriptionService) transcribeChunked(ctx context.Context, audio []byte) (types.Transcription, error) {
	chunks := [][]byte{audio}
	if split, err := s.mediaClient.Split(ctx, audio); err == nil && len(split) > 1 {
		chunks = split
	}

	result := types.Transcription{}
	offset := 0.0

	for i, chunk := range chunks {
		part, err := s.transcriptionClient.Transcribe(ctx, chunk, "mp3")
		if err != nil {
			return types.Transcription{}, fmt.Errorf("chunk %d/%d: %w", i+1, len(chunks), err)
		}

		part.BuildSegments()
		part.BuildSpeakers()
		result.AppendChunk(part, offset, i)

		offset += part.Duration
	}

	log.Printf("transcribed %d chunk(s), %.1fs total", len(chunks), result.Duration)
	return result, nil
}

// StartRecording creates the recording row and kicks the pipeline off in the
// background. The background context is detached from the request context so
// that a client disconnect does not abort processing.
func (s *transcriptionService) StartRecording(ctx context.Context, file types.Recording) (uuid.UUID, error) {
	recordingID, err := s.dbRepository.InsertRecording(ctx, file)
	if err != nil {
		return uuid.Nil, err
	}

	// Detach from the request so completion does not depend on the client
	// staying connected, but still bound the total runtime.
	bgCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), processingTimeout)

	go func() {
		defer cancel()

		if err := s.ProcessRecording(bgCtx, file, recordingID); err != nil {
			log.Printf("recording %s failed: %v", recordingID, err)
			// Status updates must survive an aborted pipeline context, so write the
			// failure state without its cancellation.
			failCtx, failCancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
			defer failCancel()
			if updateErr := s.dbRepository.UpdateRecordingStatus(failCtx, recordingID, types.RecordingFailed); updateErr != nil {
				log.Printf("recording %s: failed to set status to %q: %v", recordingID, types.RecordingFailed, updateErr)
			}
		}
	}()

	return recordingID, nil
}

// ProcessRecording runs the pipeline: upload to storage, transcribe, embed and
// match speakers, summarize, then persist the conversation. Status transitions
// follow the order:
//
//	pending -> transcribing -> embedding -> assigning_speakers -> summarizing -> completed
//
// Each transition is written when the corresponding step completes. Any error
// leaves the recording in the status of the step that failed; StartRecording is
// responsible for marking it failed.
func (s *transcriptionService) ProcessRecording(ctx context.Context, file types.Recording, recordingID uuid.UUID) error {
	// Normalise to a consistent audio-only format first. This strips video from
	// screen-recorded meetings and keeps both the stored file and the
	// transcription payload small.
	if err := s.updateStatus(ctx, recordingID, types.RecordingTranscribing); err != nil {
		return err
	}

	audio, err := s.mediaClient.Normalize(ctx, file.FileData, file.Extension)
	if err != nil {
		return fmt.Errorf("normalize audio: %w", err)
	}
	// The normalised audio is what gets stored and played back, so use an mp3
	// extension for the object key regardless of the uploaded container.
	normalized := file
	normalized.FileData = audio
	normalized.Extension = "mp3"

	if err := s.storageClient.InsertRecording(ctx, audio, normalized.UserPath, normalized.FileName, normalized.Extension); err != nil {
		return fmt.Errorf("store recording: %w", err)
	}
	// The stored object is the normalised mp3, so the row must point at it for
	// playback to resolve the right key.
	if err := s.dbRepository.UpdateRecordingExtension(ctx, recordingID, normalized.Extension); err != nil {
		return err
	}

	transcription, err := s.transcribeChunked(ctx, audio)
	if err != nil {
		return fmt.Errorf("transcribe: %w", err)
	}

	if err := s.updateStatus(ctx, recordingID, types.RecordingEmbedding); err != nil {
		return err
	}

	embeddings, err := s.embeddingClient.EmbedSpeakers(ctx, audio, transcription.Speakers)
	if err != nil {
		return fmt.Errorf("embed speakers: %w", err)
	}

	if err := s.updateStatus(ctx, recordingID, types.RecordingAssigningSpeakers); err != nil {
		return err
	}

	mappedSpeakers, err := s.mapSpeakers(ctx, file.UserID, embeddings.Data)
	if err != nil {
		return err
	}

	// Attach the resolved speaker ids to the segments before persisting them.
	// A label without a mapping means that speaker's embedding failed; fall back
	// to an anonymous speaker rather than discarding an otherwise good meeting.
	for i := range transcription.Segments {
		speakerID, ok := mappedSpeakers[transcription.Segments[i].Speaker]
		if !ok {
			speakerID, err = s.dbRepository.InsertAnonymousSpeaker(ctx, file.UserID, nil)
			if err != nil {
				return fmt.Errorf("create anonymous speaker for label %q: %w", transcription.Segments[i].Speaker, err)
			}
			log.Printf("recording %s: label %q had no embedding, created anonymous speaker %s", recordingID, transcription.Segments[i].Speaker, speakerID)
			mappedSpeakers[transcription.Segments[i].Speaker] = speakerID
		}
		transcription.Segments[i].SpeakerID = speakerID
	}

	if err := s.updateStatus(ctx, recordingID, types.RecordingSummaizing); err != nil {
		return err
	}

	summary, err := s.summaryClient.GetSummary(ctx, transcription)
	if err != nil {
		return fmt.Errorf("summarize: %w", err)
	}

	// Persist the conversation and its segments atomically.
	if err := s.dbRepository.WithTx(ctx, func(txCtx context.Context) error {
		conversationID, err := s.dbRepository.InsertCoversation(txCtx, file.Title, transcription, recordingID, summary, file.MeetingAt)
		if err != nil {
			return fmt.Errorf("insert conversation: %w", err)
		}
		if err := s.dbRepository.InsertSegments(txCtx, transcription.Segments, conversationID); err != nil {
			return fmt.Errorf("insert segments: %w", err)
		}
		return nil
	}); err != nil {
		return err
	}

	return s.updateStatus(ctx, recordingID, types.RecordingCompleted)
}

// mapSpeakers matches each speaker embedding to a known speaker, creating an
// anonymous speaker when there is no existing match.
func (s *transcriptionService) mapSpeakers(ctx context.Context, userID uuid.UUID, embeddings []types.EmbeddingResult) (map[string]uuid.UUID, error) {
	mapped := make(map[string]uuid.UUID, len(embeddings))

	for _, embedding := range embeddings {
		speaker, err := s.dbRepository.GetBestMatchingSpeaker(ctx, userID, embedding.Embedding)
		if err != nil {
			if !errors.Is(err, pgx.ErrNoRows) {
				return nil, fmt.Errorf("find matching speaker for %q: %w", embedding.SpeakerLabel, err)
			}

			newID, insertErr := s.dbRepository.InsertAnonymousSpeaker(ctx, userID, embedding.Embedding)
			if insertErr != nil {
				return nil, fmt.Errorf("create anonymous speaker for %q: %w", embedding.SpeakerLabel, insertErr)
			}
			mapped[embedding.SpeakerLabel] = newID
			continue
		}

		if err := s.dbRepository.PurgeEmbeddings(ctx, speaker.ID); err != nil {
			return nil, fmt.Errorf("prune embeddings for speaker %s: %w", speaker.ID, err)
		}
		mapped[embedding.SpeakerLabel] = speaker.ID
	}

	return mapped, nil
}

func (s *transcriptionService) updateStatus(ctx context.Context, recordingID uuid.UUID, status types.RecordingStatus) error {
	if err := s.dbRepository.UpdateRecordingStatus(ctx, recordingID, status); err != nil {
		return fmt.Errorf("set recording status to %q: %w", status, err)
	}
	return nil
}
