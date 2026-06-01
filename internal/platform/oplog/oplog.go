package oplog

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/lucky-aeon/agentx/plugin-helper/internal/platform/xlog"
)

const defaultFileName = "operations.log"

type Level string

const (
	LevelDebug Level = "debug"
	LevelInfo  Level = "info"
	LevelWarn  Level = "warn"
	LevelError Level = "error"
)

type Event struct {
	ID           string                 `json:"id"`
	Timestamp    time.Time              `json:"timestamp"`
	Level        Level                  `json:"level"`
	Action       string                 `json:"action"`
	Message      string                 `json:"message"`
	Source       string                 `json:"source,omitempty"`
	WorkspaceID  string                 `json:"workspace_id,omitempty"`
	SessionID    string                 `json:"session_id,omitempty"`
	ResourceType string                 `json:"resource_type,omitempty"`
	ResourceID   string                 `json:"resource_id,omitempty"`
	ActorID      string                 `json:"actor_id,omitempty"`
	Error        string                 `json:"error,omitempty"`
	Detail       map[string]interface{} `json:"detail,omitempty"`
}

type Query struct {
	WorkspaceID string
	Limit       int
}

type Store interface {
	List(context.Context, Query) ([]Event, error)
}

type recordStore interface {
	Store
	record(context.Context, Event)
}

type XLogSink struct {
	recorder recordStore
}

func NewXLogSink(store Store) *XLogSink {
	recorder, _ := store.(recordStore)
	return &XLogSink{recorder: recorder}
}

func (s *XLogSink) Write(entry xlog.Entry) {
	if s == nil || s.recorder == nil {
		return
	}
	if asString(entry.Fields["log_type"]) != "operation" {
		return
	}
	event := Event{
		ID:           asString(entry.Fields["event_id"]),
		Timestamp:    entry.Timestamp,
		Level:        Level(entry.Level),
		Action:       asString(entry.Fields["action"]),
		Message:      entry.Message,
		Source:       entry.Logger,
		WorkspaceID:  asString(entry.Fields["workspace_id"]),
		SessionID:    asString(entry.Fields["session_id"]),
		ResourceType: asString(entry.Fields["resource_type"]),
		ResourceID:   asString(entry.Fields["resource_id"]),
		ActorID:      asString(entry.Fields["actor_id"]),
		Error:        asString(entry.Fields["error"]),
		Detail:       detailFromFields(entry.Fields),
	}
	if event.ID == "" {
		event.ID = asString(entry.Fields["id"])
	}
	s.recorder.record(context.Background(), event)
}

type NoopStore struct{}

func (NoopStore) List(context.Context, Query) ([]Event, error) {
	return nil, nil
}

type FileRecorder struct {
	path string
	mu   sync.Mutex
}

func NewFileRecorder(workspacePath string) (*FileRecorder, error) {
	if workspacePath == "" {
		return nil, errors.New("workspace path is empty")
	}
	dir := filepath.Join(workspacePath, "logs")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	return &FileRecorder{path: filepath.Join(dir, defaultFileName)}, nil
}

func (r *FileRecorder) record(_ context.Context, event Event) {
	if r == nil {
		return
	}
	normalizeEvent(&event)

	line, err := json.Marshal(event)
	if err != nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	file, err := os.OpenFile(r.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer file.Close()
	_, _ = file.Write(append(line, '\n'))
}

func (r *FileRecorder) List(ctx context.Context, q Query) ([]Event, error) {
	if r == nil {
		return nil, nil
	}
	limit := q.Limit
	if limit <= 0 {
		limit = 200
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	file, err := os.Open(r.path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	items := make([]Event, 0, limit)
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		var event Event
		if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
			continue
		}
		if q.WorkspaceID != "" && event.WorkspaceID != q.WorkspaceID {
			continue
		}
		items = append(items, event)
		if len(items) > limit {
			copy(items[0:], items[len(items)-limit:])
			items = items[:limit]
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	sort.SliceStable(items, func(i, j int) bool {
		return items[i].Timestamp.After(items[j].Timestamp)
	})
	return items, nil
}

func detailFromFields(fields map[string]interface{}) map[string]interface{} {
	detail := map[string]interface{}{}
	for k, v := range fields {
		switch k {
		case "log_type", "event_id", "id", "action", "workspace_id", "session_id", "resource_type", "resource_id", "actor_id", "error":
			continue
		default:
			detail[k] = v
		}
	}
	if len(detail) == 0 {
		return nil
	}
	return detail
}

func asString(v interface{}) string {
	switch typed := v.(type) {
	case string:
		return typed
	case fmt.Stringer:
		return typed.String()
	default:
		return ""
	}
}
