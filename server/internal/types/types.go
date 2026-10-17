package types

import (
	"log"
	"slices"

	"github.com/google/uuid"
)

type Recording struct {
	Title     string
	FileName  string
	Extension string
	FileData  []byte
	UserPath  string
}

type RecordingStatus string

const (
	RecordingTranscribing      RecordingStatus = "transcribing"
	RecordingEmbedding         RecordingStatus = "embedding"
	RecordingSummaizing        RecordingStatus = "summarizing"
	RecordingAssigningSpeakers RecordingStatus = "assigning_speakers"
	RecordingCompleted         RecordingStatus = "completed"
	RecordingFailed            RecordingStatus = "failed"
)

type TranscriptionUsage struct {
	Seconds float64 `json:"seconds"`
	Cost    float64 `json:"cost"`
}

type TranscriptionWord struct {
	Word         string  `json:"word"`
	Start        float64 `json:"start"`
	End          float64 `json:"end"`
	Speaker      int     `json:"speaker"`
	Confidence   float64 `json:"confidence"`
	SpeakerLabel string  `json:"speaker_label"`
}

type Transcription struct {
	Text               string              `json:"text"`
	Usage              TranscriptionUsage  `json:"usage"`
	Task               string              `json:"task"`
	Language           string              `json:"language"`
	LanguageConfidence float64             `json:"language_confidence"`
	Duration           float64             `json:"duration"`
	Words              []TranscriptionWord `json:"words"`
	Segments           []TranscriptionSegment
	Speakers           []SpeakerData
}

type TranscriptionSegment struct {
	Speaker   string
	Start     float64
	End       float64
	Text      string
	Duration  float64
	SpeakerID uuid.UUID
}

type TranscriptionSpeaker struct {
	Label string  `json:"speaker_label"`
	Start float64 `json:"start"`
	End   float64 `json:"end"`
}

func (t *Transcription) BuildSegments() {
	if len(t.Words) == 0 {
		return
	}

	var segments []TranscriptionSegment

	current := TranscriptionSegment{
		Speaker: t.Words[0].SpeakerLabel,
		Start:   t.Words[0].Start,
		End:     t.Words[0].End,
		Text:    t.Words[0].Word,
	}

	for _, word := range t.Words[1:] {
		if word.SpeakerLabel != current.Speaker {
			current.Duration = current.End - current.Start
			segments = append(segments, current)

			current = TranscriptionSegment{
				Speaker: word.SpeakerLabel,
				Start:   word.Start,
				End:     word.End,
				Text:    word.Word,
			}
			continue
		}

		current.Text += " " + word.Word
		current.End = word.End
	}

	current.Duration = current.End - current.Start
	segments = append(segments, current)

	log.Printf("Built segments: %v", segments)
	t.Segments = segments
}

// BuildSpeakers Must be run after segments are built or it will return empty
func (t *Transcription) BuildSpeakers() {
	var speakers []SpeakerData
	var speakersSeen []string
	for _, segment := range t.Segments {
		if segment.Duration > 4 && !slices.Contains(speakersSeen, segment.Speaker) {
			speakers = append(speakers, SpeakerData{
				SpeakerLabel: segment.Speaker,
				Start:        segment.Start,
				End:          segment.End,
			})
		}
	}
	log.Printf("Built speakers: %v", speakers)
	t.Speakers = speakers
}

type EmbeddingResult struct {
	SpeakerLabel string    `json:"speaker_label"`
	Embedding    []float32 `json:"embedding"`
}

type EmbeddingResponse struct {
	Data []EmbeddingResult `json:"data"`
}

type SpeakerData struct {
	SpeakerLabel string  `json:"speaker_label"`
	Start        float64 `json:"start"`
	End          float64 `json:"end"`
}

type EmbeddingRequest struct {
	File     []byte        `json:"file"`
	Speakers []SpeakerData `json:"speakers"`
}

type IdentifiedSpeaker struct {
	ID              uuid.UUID
	SimilarityScore float64
}
