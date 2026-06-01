package oplog

import (
	"context"
	"testing"
	"time"
)

func TestFileRecorderListFiltersAndTailsWorkspaceEvents(t *testing.T) {
	recorder, err := NewFileRecorder(t.TempDir())
	if err != nil {
		t.Fatalf("NewFileRecorder() error = %v", err)
	}
	ctx := context.Background()
	base := time.Date(2026, 6, 1, 1, 0, 0, 0, time.UTC)

	recorder.record(ctx, Event{ID: "1", Timestamp: base, Level: LevelInfo, Action: "workspace.create", Message: "created", WorkspaceID: "alpha"})
	recorder.record(ctx, Event{ID: "2", Timestamp: base.Add(time.Minute), Level: LevelError, Action: "session.request_failed", Message: "failed", WorkspaceID: "beta", Error: "boom"})
	recorder.record(ctx, Event{ID: "3", Timestamp: base.Add(2 * time.Minute), Level: LevelInfo, Action: "session.request", Message: "request", WorkspaceID: "alpha"})

	items, err := recorder.List(ctx, Query{WorkspaceID: "alpha", Limit: 1})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(items))
	}
	if items[0].ID != "3" {
		t.Fatalf("items[0].ID = %q, want %q", items[0].ID, "3")
	}
	if items[0].WorkspaceID != "alpha" {
		t.Fatalf("items[0].WorkspaceID = %q, want alpha", items[0].WorkspaceID)
	}
}
