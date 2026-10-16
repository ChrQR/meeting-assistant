package types

type Recording struct {
	Title     string
	FileName  string
	Extension string
	FileData  []byte
	UserPath  string
}

type RecordingStatus string

const (
	RecordingTranscribing RecordingStatus = "transcribing"
	RecordingEmbedding    RecordingStatus = "embedding"
	RecordingCompleted    RecordingStatus = "completed"
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
}

type TranscriptionSegment struct {
	Speaker  string
	Start    float64
	End      float64
	Text     string
	Duration float64
}

func (t *Transcription) BuildSegments() {
	var segment TranscriptionSegment
	segments := make([]TranscriptionSegment, 0)
	for i, word := range t.Words {
		if word.SpeakerLabel != segment.Speaker && i != 0 {
			segments = append(segments, segment)
			segment = TranscriptionSegment{}
			segment.Speaker = word.SpeakerLabel
			segment.Start = word.Start
			segment.End = word.End
			segment.Text = word.Word
		}
		segment.Text += word.Word
		segment.End = word.End
		segment.Duration = segment.End - segment.Start
	}
	t.Segments = segments
}

type EmbeddingResult struct {
	SpeakerLabel string    `json:"speaker_label"`
	Embedding    []float64 `json:"embedding"`
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
