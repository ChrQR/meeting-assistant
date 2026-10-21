package client

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

const (
	// mediaChunkDuration is how long each transcription request covers. ElevenLabs
	// (via OpenRouter) timed out at ~65 minutes in one request but handled 60
	// minutes comfortably, so this leaves a wide margin.
	mediaChunkDuration = 30 * time.Minute

	normalizeTimeout = 30 * time.Minute
	chunkTimeout     = 10 * time.Minute
)

// MediaClient prepares uploaded recordings for transcription: it normalises
// arbitrary containers (MP4/MOV/M4A/WAV/...) to a consistent audio format and
// splits long audio into chunks for separate transcription requests.
type MediaClient interface {
	// Normalize converts the uploaded file to a mono 16kHz MP3 audio file.
	Normalize(ctx context.Context, fileBytes []byte, extension string) ([]byte, error)
	// Split divides normalised audio into contiguous chunks of at most
	// mediaChunkDuration. Chunks are returned in order.
	Split(ctx context.Context, audio []byte) ([][]byte, error)
	// Available reports whether the required ffmpeg binaries are installed.
	Available() bool
}

type mediaClient struct {
	ffmpegPath string
}

func NewMediaClient() MediaClient {
	return &mediaClient{ffmpegPath: "ffmpeg"}
}

func (c *mediaClient) Available() bool {
	for _, binary := range []string{c.ffmpegPath, "ffprobe"} {
		if _, err := exec.LookPath(binary); err != nil {
			return false
		}
	}
	return true
}

// Normalize decodes any supported container and re-encodes audio only, dropping
// video. This both shrinks the payload (Teams exports are mostly video) and
// produces a format the transcription provider accepts.
func (c *mediaClient) Normalize(ctx context.Context, fileBytes []byte, extension string) ([]byte, error) {
	if len(fileBytes) == 0 {
		return nil, fmt.Errorf("empty recording")
	}

	ctx, cancel := context.WithTimeout(ctx, normalizeTimeout)
	defer cancel()

	extension = normalizeExtension(extension)
	tempDir, err := os.MkdirTemp("", "meeting-media-*")
	if err != nil {
		return nil, fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tempDir)

	inputPath := filepath.Join(tempDir, "input."+extension)
	outputPath := filepath.Join(tempDir, "output.mp3")

	if err := os.WriteFile(inputPath, fileBytes, 0o600); err != nil {
		return nil, fmt.Errorf("write temp input: %w", err)
	}

	cmd := exec.CommandContext(ctx, c.ffmpegPath,
		"-nostdin",
		"-hide_banner",
		"-loglevel", "error",
		"-i", inputPath,
		"-vn",                // drop video streams (Teams/Zoom recordings)
		"-ac", "1",           // mono: speaker embeddings and STT expect a single channel
		"-ar", "16000",       // 16kHz is sufficient for speech and halves the size
		"-c:a", "libmp3lame", // MP3 is accepted by the transcription provider
		"-b:a", "32k",
		"-y",
		outputPath,
	)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("convert audio: %w: %s", err, truncate(stderr.Bytes(), 500))
	}

	audio, err := os.ReadFile(outputPath)
	if err != nil {
		return nil, fmt.Errorf("read converted audio: %w", err)
	}
	if len(audio) == 0 {
		return nil, fmt.Errorf("converted audio is empty")
	}
	return audio, nil
}

// Split cuts normalised audio into fixed-length chunks without re-encoding.
func (c *mediaClient) Split(ctx context.Context, audio []byte) ([][]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, chunkTimeout)
	defer cancel()

	tempDir, err := os.MkdirTemp("", "meeting-chunks-*")
	if err != nil {
		return nil, fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tempDir)

	inputPath := filepath.Join(tempDir, "audio.mp3")
	if err := os.WriteFile(inputPath, audio, 0o600); err != nil {
		return nil, fmt.Errorf("write temp audio: %w", err)
	}

	chunkPattern := filepath.Join(tempDir, "chunk-%03d.mp3")
	cmd := exec.CommandContext(ctx, c.ffmpegPath,
		"-nostdin",
		"-hide_banner",
		"-loglevel", "error",
		"-i", inputPath,
		"-f", "segment",
		"-segment_time", fmt.Sprintf("%d", int(mediaChunkDuration.Seconds())),
		"-c", "copy", // no re-encode: segments stay decodable on their own
		"-y",
		chunkPattern,
	)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("split audio: %w: %s", err, truncate(stderr.Bytes(), 500))
	}

	paths, err := filepath.Glob(filepath.Join(tempDir, "chunk-*.mp3"))
	if err != nil {
		return nil, err
	}
	if len(paths) == 0 {
		return nil, fmt.Errorf("splitting produced no chunks")
	}

	chunks := make([][]byte, 0, len(paths))
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read chunk %s: %w", filepath.Base(path), err)
		}
		chunks = append(chunks, data)
	}
	return chunks, nil
}

// normalizeExtension strips a leading dot and rejects values that cannot be used
// safely as a file extension.
func normalizeExtension(extension string) string {
	extension = filepath.Ext("x." + extension)
	extension = extension[1:]
	if extension == "" {
		return "bin"
	}
	return extension
}
