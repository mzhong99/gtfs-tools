package platform

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type FeedFrameMeta struct {
	SequenceNumber int       `json:"seq"`
	FeedID         string    `json:"feed_id"`
	CapturedAt     time.Time `json:"captured_at"`
	SHA256         string    `json:"sha256"`
	Source         string    `json:"source,omitempty"`
	PayloadPath    string    `json:"payload,omitempty"`

	HTTP *struct {
		Status        int   `json:"status"`
		DurationMs    int64 `json:"duration_ms"`
		ContentLength int64 `json:"content_length"`
	} `json:"http,omitempty"`

	Error string `json:"error,omitempty"`
}

type FeedFrame struct {
	FeedID     string
	CapturedAt time.Time
	Status     int
	Source     string
	Body       []byte
	SHA256     [32]byte
}

func (frame FeedFrame) String() string {
	bodyLen := len(frame.Body)

	var timestamp string
	if !frame.CapturedAt.IsZero() {
		timestamp = frame.CapturedAt.Format(time.RFC3339Nano)
	}

	return fmt.Sprintf(
		"FeedFrame{feed=%q source=%q status=%d ts=%s sha256=%s body=%dB}",
		frame.FeedID,
		frame.Source,
		frame.Status,
		timestamp,
		hex.EncodeToString(frame.SHA256[:4]),
		bodyLen,
	)
}

type ToolInfo struct {
	Name    string `json:"name"`
	Version string `json:"version,omitempty"`
	GitSHA  string `json:"git_sha,omitempty"`
}

type RecordingHeader struct {
	SchemaVersion int    `json:"schema_version"`
	RecordingName string `json:"recording_name,omitempty"`
	RecordingUID  string `json:"recording_uid"`

	Format    string     `json:"format"`
	CreatedAt time.Time  `json:"created_at"`
	TimeZone  string     `json:"time_zone,omitempty"`
	Tool      ToolInfo   `json:"tool"`
	Feeds     []FeedSpec `json:"feeds"`
}

type RecordingHeaderOptions struct {
	RecordingName string
	CreatedAt     time.Time
	TimeZone      string
	Tool          ToolInfo
}

type FeedSpec struct {
	FeedID      string  `json:"feed_id"`
	URL         string  `json:"url,omitempty"`
	PollSeconds float64 `json:"poll_seconds,omitempty"`
}

type FeedRecordingWriter struct {
	header     RecordingHeader
	rootDir    string
	payloadDir string

	framesFile   *os.File
	framesWriter *bufio.Writer

	sequenceNumber int
	lock           sync.Mutex
	closed         bool
}

type FeedRecordingReader struct {
	header     RecordingHeader
	rootDir    string
	payloadDir string

	framesFile *os.File
	scanner    *bufio.Scanner

	sequenceNumber int
}

func NewToolInfo(name string) ToolInfo {
	return ToolInfo{Name: name, GitSHA: GitCommit, Version: Version}
}

func GenerateUID(nbytes int) string {
	buffer := make([]byte, nbytes)
	if _, err := rand.Read(buffer); err != nil {
		panic(err)
	}
	return hex.EncodeToString(buffer)
}

func BuildRecordingHeader(opts RecordingHeaderOptions, feeds []FeedSpec) (RecordingHeader, error) {
	if len(feeds) == 0 {
		return RecordingHeader{}, fmt.Errorf("recording must include at least one feed")
	}

	seen := make(map[string]bool, len(feeds))
	for _, feed := range feeds {
		if feed.FeedID == "" {
			return RecordingHeader{}, fmt.Errorf("feed_spec missing feed_id")
		}
		if _, ok := seen[feed.FeedID]; ok {
			return RecordingHeader{}, fmt.Errorf("duplicate feed_id %q in recording header", feed.FeedID)
		}
		seen[feed.FeedID] = true

		if feed.URL == "" {
			return RecordingHeader{}, fmt.Errorf("feed_spec %q missing url", feed.FeedID)
		}
		if feed.PollSeconds <= 0 {
			return RecordingHeader{}, fmt.Errorf("feed_spec %q poll_sec must be > 0", feed.FeedID)
		}
	}

	createdAt := opts.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now()
	}

	tool := opts.Tool
	if tool.Name == "" {
		tool.Name = "unknown"
	}

	return RecordingHeader{
		SchemaVersion: 1,
		Format:        "gtfs-rt",
		RecordingName: opts.RecordingName,
		RecordingUID:  GenerateUID(16),
		CreatedAt:     opts.CreatedAt,
		TimeZone:      opts.TimeZone,
		Tool:          tool,
		Feeds:         feeds,
	}, nil
}

func CreateFeedRecording(
	recordingDir string,
	feeds []FeedSpec,
	opts RecordingHeaderOptions,
) (*FeedRecordingWriter, error) {
	header, err := BuildRecordingHeader(opts, feeds)
	if err != nil {
		return nil, err
	}

	payloadDir := filepath.Join(recordingDir, "payloads")
	if err := os.MkdirAll(payloadDir, 0o755); err != nil {
		return nil, fmt.Errorf("Could not create %s: %w", recordingDir, err)
	}

	headerPath := filepath.Join(recordingDir, "recording.json")
	if err := writeJSONFileAtomic(headerPath, header, 0o644); err != nil {
		return nil, fmt.Errorf("write recording.json: %w", err)
	}

	framesPath := filepath.Join(recordingDir, "frames.jsonl")
	framesFile, err := os.OpenFile(framesPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, fmt.Errorf("open frames.jsonl: %w", err)
	}

	return &FeedRecordingWriter{
		header:         header,
		rootDir:        recordingDir,
		payloadDir:     payloadDir,
		framesFile:     framesFile,
		framesWriter:   bufio.NewWriterSize(framesFile, 256*1024),
		lock:           sync.Mutex{},
		sequenceNumber: 0,
	}, nil
}

func (writer *FeedRecordingWriter) Header() RecordingHeader {
	return writer.header
}

func (writer *FeedRecordingWriter) RootDir() string {
	return writer.rootDir
}

func (writer *FeedRecordingWriter) Append(ctx context.Context, frame FeedFrame) error {
	_ = ctx // for now we're not using the ctx, this is for future long write cancels

	writer.lock.Lock()
	defer writer.lock.Unlock()

	if writer.closed {
		return fmt.Errorf("append on closed recording writer")
	}

	writer.sequenceNumber++

	meta := FeedFrameMeta{
		SequenceNumber: writer.sequenceNumber,
		FeedID:         frame.FeedID,
		CapturedAt:     frame.CapturedAt,
		Source:         frame.Source,
		SHA256:         hex.EncodeToString(frame.SHA256[:]),

		// TODO: Fill frame status here
	}

	if len(frame.Body) > 0 {
		payloadPathRel := filepath.Join(
			"payloads",
			fmt.Sprintf("%06d.pb", writer.sequenceNumber),
		)
		payloadPathAbs := filepath.Join(writer.rootDir, payloadPathRel)
		if err := writeFileAtomic(payloadPathAbs, frame.Body, 0o644); err != nil {
			return fmt.Errorf("write payload: %w", err)
		}
		meta.PayloadPath = payloadPathRel
	}

	line, err := json.Marshal(&meta)
	if err != nil {
		return fmt.Errorf("marshal frame meta: %w", err)
	}
	if _, err := writer.framesWriter.Write(line); err != nil {
		return fmt.Errorf("write frames.jsonl: %w", err)
	}
	if err := writer.framesWriter.WriteByte('\n'); err != nil {
		return fmt.Errorf("write frames.jsonl newline: %w", err)
	}

	if err := writer.framesWriter.Flush(); err != nil {
		return fmt.Errorf("flush frames.jsonl: %w", err)
	}

	return nil
}

func (writer *FeedRecordingWriter) Close() error {
	return nil
}

func OpenFeedRecording(rootDir string) (*FeedRecordingReader, error) {
	return nil, nil
}

func (reader *FeedRecordingReader) Reset() {
	reader.sequenceNumber = 0
}

func (reader *FeedRecordingReader) Next(ctx context.Context) (FeedFrame, error) {
	return FeedFrame{}, nil
}

func writeJSONFileAtomic(path string, v any, perm os.FileMode) error {
	buffer, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	buffer = append(buffer, '\n')

	return writeFileAtomic(path, buffer, perm)
}

func writeFileAtomic(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	base := filepath.Base(path)
	tmp := filepath.Join(dir, "."+base+".tmp")

	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, perm)
	if err != nil {
		return err
	}
	_, werr := f.Write(data)
	cerr := f.Close()

	if werr != nil {
		_ = os.Remove(tmp)
		return werr
	}
	if cerr != nil {
		_ = os.Remove(tmp)
		return cerr
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}
