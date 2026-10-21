package types

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Recording struct {
	UserID    uuid.UUID
	Title     string
	FileName  string
	Extension string
	FileData  []byte
	UserPath  string
	// MeetingAt is when the meeting took place, if supplied on upload.
	MeetingAt *time.Time
}

type RecordingStatus string

const (
	RecordingPending           RecordingStatus = "pending"
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

// AppendChunk merges the transcript of a later audio chunk into t.
//
// Every word and segment in the chunk is shifted by offset seconds so it lines
// up with the original recording. Speaker labels are prefixed with the chunk
// index because diarization happens per request: "speaker_0" in the second
// chunk is not necessarily the person called "speaker_0" in the first. The
// voice embeddings later resolve those labels back to real speakers.
func (t *Transcription) AppendChunk(chunk Transcription, offset float64, chunkIndex int) {
	labelPrefix := fmt.Sprintf("chunk%d_", chunkIndex)

	rename := func(label string) string {
		if label == "" {
			return labelPrefix + "unknown"
		}
		return labelPrefix + label
	}

	for _, word := range chunk.Words {
		word.Start += offset
		word.End += offset
		word.SpeakerLabel = rename(word.SpeakerLabel)
		t.Words = append(t.Words, word)
	}

	for _, segment := range chunk.Segments {
		segment.Start += offset
		segment.End += offset
		segment.Speaker = rename(segment.Speaker)
		t.Segments = append(t.Segments, segment)
	}

	// Speaker samples drive the voice embeddings. Chunk labels are prefixed, so
	// the samples can simply be appended.
	for _, speaker := range chunk.Speakers {
		speaker.SpeakerLabel = rename(speaker.SpeakerLabel)
		speaker.Start += offset
		speaker.End += offset
		t.Speakers = append(t.Speakers, speaker)
	}

	if chunk.Text != "" {
		if t.Text != "" {
			t.Text += " "
		}
		t.Text += chunk.Text
	}
	if chunk.Duration > 0 {
		t.Duration = offset + chunk.Duration
	}

	// Accumulate usage so the conversation cost reflects every request.
	t.Usage.Seconds += chunk.Usage.Seconds
	t.Usage.Cost += chunk.Usage.Cost

	if t.Language == "" {
		t.Language = chunk.Language
		t.LanguageConfidence = chunk.LanguageConfidence
	}
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

	log.Printf("Built %d segments across %d speakers", len(segments), t.countSpeakers(segments))
	t.Segments = segments
}

// BuildSpeakers derives the speakers that should be voice-embedded from the
// segments. A speaker's sample is the start of their first sufficiently long
// segment; speakers that only ever speak briefly fall back to their longest
// segment so they are never dropped (dropping them would leave segments
// referring to a speaker that was never embedded).
func (t *Transcription) BuildSpeakers() {
	seen := make(map[string]bool)
	longest := make(map[string]TranscriptionSegment)
	sample := make(map[string]TranscriptionSegment)
	order := make([]string, 0)

	for _, segment := range t.Segments {
		label := segment.Speaker
		if label == "" {
			continue
		}

		if !seen[label] {
			seen[label] = true
			order = append(order, label)
		}

		// Prefer the speaker's first segment long enough to produce a reliable
		// voice embedding.
		if _, ok := sample[label]; !ok && segment.Duration > 4 {
			sample[label] = segment
		}

		// Track the longest segment as a fallback for speakers who never speak
		// long enough on their own.
		if current, ok := longest[label]; !ok || segment.Duration > current.Duration {
			longest[label] = segment
		}
	}

	speakers := make([]SpeakerData, 0, len(order))
	for _, label := range order {
		segment, ok := sample[label]
		if !ok {
			segment = longest[label]
		}
		if segment.Duration <= 0 {
			continue
		}
		speakers = append(speakers, SpeakerData{
			SpeakerLabel: label,
			Start:        segment.Start,
			End:          segment.End,
		})
	}

	log.Printf("Built speakers (%d): %v", len(speakers), speakerLabels(speakers))
	t.Speakers = speakers
}

func (t *Transcription) countSpeakers(segments []TranscriptionSegment) int {
	seen := make(map[string]struct{}, len(segments))
	for _, segment := range segments {
		seen[segment.Speaker] = struct{}{}
	}
	return len(seen)
}

func speakerLabels(speakers []SpeakerData) []string {
	labels := make([]string, 0, len(speakers))
	for _, speaker := range speakers {
		labels = append(labels, speaker.SpeakerLabel)
	}
	return labels
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

type SelectRecording struct {
	ID        uuid.UUID
	Extension string
	Filename  string
	Status    RecordingStatus
}

// RecordingFile is everything needed to locate and stream a stored recording.
type RecordingFile struct {
	ID        uuid.UUID
	Path      string
	Filename  string
	Extension string
}

type ConversationSummary struct {
	ID                  uuid.UUID
	Title               string
	Summary             *string
	Length              float64
	Cost                float64
	Language            string
	CreatedAt           time.Time
	MeetingAt           *time.Time
	RecordingID         uuid.UUID
	RecordingStatus     string
	SpeakerCount        int
	UnknownSpeakerCount int
}

type ConversationSegment struct {
	ID          uuid.UUID
	Text        string
	StartTime   float64
	EndTime     float64
	Duration    float64
	SpeakerID   uuid.UUID
	SpeakerName *string
	Ignored     bool
}

type Conversation struct {
	ConversationSummary
	Segments []ConversationSegment
}

type Speaker struct {
	ID                uuid.UUID
	Name              *string
	Email             *string
	CompanyID         *uuid.UUID
	CompanyName       *string
	UserID            *uuid.UUID
	UserEmail         *string
	EmbeddingCount    int
	SegmentCount      int
	TotalSpeakingTime float64
	ConversationCount int
	CreatedAt         time.Time
}

type SpeakerEmbedding struct {
	ID        uuid.UUID
	CreatedAt time.Time
}

type SpeakerSegment struct {
	ID                uuid.UUID
	Text              string
	StartTime         float64
	EndTime           float64
	ConversationID    uuid.UUID
	ConversationTitle string
}

type SpeakerDetail struct {
	Speaker
	Embeddings []SpeakerEmbedding
	Segments   []SpeakerSegment
}

type Company struct {
	ID           uuid.UUID
	Name         string
	SpeakerCount int
	CreatedAt    time.Time
}

// CreateSpeaker describes a named speaker to be inserted.
type CreateSpeaker struct {
	Name      string
	Email     *string
	CompanyID *uuid.UUID
}

// UpdateSpeaker describes a partial update. Nil fields are left untouched.
type UpdateSpeaker struct {
	Name         *string
	Email        *string
	CompanyID    *uuid.UUID
	ClearCompany bool
	// UserID links the speaker to an application user.
	UserID    *uuid.UUID
	ClearUser bool
}

// AssignSpeaker resolves an unknown speaker. Exactly one of TargetSpeakerID or
// Name must be set.
type AssignSpeaker struct {
	TargetSpeakerID *uuid.UUID
	Name            *string
	Email           *string
	CompanyID       *uuid.UUID
}

// ParseMeetingAt accepts a date (YYYY-MM-DD) or a full RFC 3339 timestamp. A
// date-only value is interpreted as local midnight.
func ParseMeetingAt(value string) (*time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}

	if parsed, err := time.Parse(time.RFC3339, value); err == nil {
		return &parsed, nil
	}

	if parsed, err := time.Parse("2006-01-02", value); err == nil {
		local := time.Date(parsed.Year(), parsed.Month(), parsed.Day(), 0, 0, 0, 0, time.Local)
		return &local, nil
	}

	return nil, fmt.Errorf("invalid meeting date %q: expected YYYY-MM-DD or RFC 3339", value)
}

// AppUser is a local user identity. External providers are linked via Account.
type AppUser struct {
	ID        uuid.UUID
	Email     string
	Name      *string
	AvatarURL *string
	CreatedAt time.Time
}

// NewUser is the profile data supplied when creating or updating a user.
type NewUser struct {
	Email     string
	Name      *string
	AvatarURL *string
}

// Account links an external identity provider to a user.
type Account struct {
	ID                uuid.UUID
	UserID            uuid.UUID
	Provider          string
	ProviderAccountID string
}
