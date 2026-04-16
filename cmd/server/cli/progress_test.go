package cli

import (
	"bytes"
	"io"
	"testing"
	"time"

	prettyprogress "github.com/jedib0t/go-pretty/v6/progress"

	"github.com/FlameInTheDark/emerald/internal/pipeline"
)

func TestPipelineProgressAdapterCreatesTrackersOnlyForStartedNodes(t *testing.T) {
	t.Parallel()

	writer := &fakeProgressWriter{}
	adapter := newPipelineProgressAdapter("demo", pipeline.FlowData{
		Nodes: []pipeline.FlowNode{
			{ID: "node-a", Data: []byte(`{"label":"Alpha"}`)},
			{ID: "node-b", Data: []byte(`{"label":"Beta"}`)},
			{ID: "node-c", Data: []byte(`{"label":"Gamma"}`)},
		},
	}, writer)

	adapter.HandleEvent(pipeline.ProgressEvent{
		Kind:        pipeline.ProgressEventExecutionStarted,
		ExecutionID: "exec-1",
	})
	adapter.HandleEvent(pipeline.ProgressEvent{
		Kind:     pipeline.ProgressEventNodeStarted,
		NodeID:   "node-b",
		NodeType: "logic:return",
	})
	adapter.HandleEvent(pipeline.ProgressEvent{
		Kind:     pipeline.ProgressEventNodeStarted,
		NodeID:   "node-a",
		NodeType: "action:shell_command",
	})

	if len(writer.trackers) != 2 {
		t.Fatalf("expected 2 active trackers, got %d", len(writer.trackers))
	}
	if writer.trackers[0].Message != "Beta (logic:return)" {
		t.Fatalf("unexpected first tracker label %q", writer.trackers[0].Message)
	}
	if writer.trackers[1].Message != "Alpha (action:shell_command)" {
		t.Fatalf("unexpected second tracker label %q", writer.trackers[1].Message)
	}
	if writer.trackers[0].Index != 0 || writer.trackers[1].Index != 1 {
		t.Fatalf("expected tracker indices to follow start order, got %d and %d", writer.trackers[0].Index, writer.trackers[1].Index)
	}
}

func TestPipelineProgressAdapterMarksCompletedAndFailedTrackers(t *testing.T) {
	t.Parallel()

	writer := &fakeProgressWriter{}
	adapter := newPipelineProgressAdapter("demo", pipeline.FlowData{
		Nodes: []pipeline.FlowNode{
			{ID: "node-a", Data: []byte(`{"label":"Alpha"}`)},
			{ID: "node-b", Data: []byte(`{"label":"Beta"}`)},
			{ID: "node-c", Data: []byte(`{"label":"Gamma"}`)},
		},
	}, writer)

	adapter.HandleEvent(pipeline.ProgressEvent{Kind: pipeline.ProgressEventExecutionStarted, ExecutionID: "exec-1"})
	adapter.HandleEvent(pipeline.ProgressEvent{Kind: pipeline.ProgressEventNodeStarted, NodeID: "node-a", NodeType: "action:shell_command"})
	adapter.HandleEvent(pipeline.ProgressEvent{Kind: pipeline.ProgressEventNodeCompleted, NodeID: "node-a", NodeType: "action:shell_command", Status: "completed"})
	adapter.HandleEvent(pipeline.ProgressEvent{Kind: pipeline.ProgressEventNodeStarted, NodeID: "node-b", NodeType: "logic:return"})
	adapter.HandleEvent(pipeline.ProgressEvent{Kind: pipeline.ProgressEventNodeCompleted, NodeID: "node-b", NodeType: "logic:return", Status: "failed"})

	if len(writer.trackers) != 2 {
		t.Fatalf("expected 2 trackers, got %d", len(writer.trackers))
	}
	if !writer.trackers[0].IsDone() || writer.trackers[0].IsErrored() {
		t.Fatalf("expected first tracker to be done without error")
	}
	if !writer.trackers[1].IsDone() || !writer.trackers[1].IsErrored() {
		t.Fatalf("expected second tracker to be done with error")
	}
	if got := writer.lastPinnedMessages(); len(got) != 4 || got[1] != "Execution ID: exec-1" || got[3] != "Done/Failed: 1/1" {
		t.Fatalf("unexpected pinned messages: %#v", got)
	}
	if len(writer.trackersByMessage("Gamma")) != 0 {
		t.Fatalf("inactive node tracker should not have been created")
	}
}

type fakeProgressWriter struct {
	trackers       []*prettyprogress.Tracker
	pinnedMessages [][]string
	rendered       bool
	stopped        bool
	output         io.Writer
}

func (w *fakeProgressWriter) AppendTracker(tracker *prettyprogress.Tracker) {
	w.trackers = append(w.trackers, tracker)
}

func (w *fakeProgressWriter) Render() {
	w.rendered = true
}

func (w *fakeProgressWriter) SetAutoStop(bool) {}

func (w *fakeProgressWriter) SetMessageLength(int) {}

func (w *fakeProgressWriter) SetOutputWriter(output io.Writer) {
	w.output = output
}

func (w *fakeProgressWriter) SetPinnedMessages(messages ...string) {
	copied := append([]string(nil), messages...)
	w.pinnedMessages = append(w.pinnedMessages, copied)
}

func (w *fakeProgressWriter) SetSortBy(prettyprogress.SortBy) {}

func (w *fakeProgressWriter) SetStyle(prettyprogress.Style) {}

func (w *fakeProgressWriter) SetTrackerLength(int) {}

func (w *fakeProgressWriter) SetUpdateFrequency(time.Duration) {}

func (w *fakeProgressWriter) Stop() {
	w.stopped = true
}

func (w *fakeProgressWriter) lastPinnedMessages() []string {
	if len(w.pinnedMessages) == 0 {
		return nil
	}
	return w.pinnedMessages[len(w.pinnedMessages)-1]
}

func (w *fakeProgressWriter) trackersByMessage(fragment string) []*prettyprogress.Tracker {
	matches := make([]*prettyprogress.Tracker, 0)
	for _, tracker := range w.trackers {
		if bytes.Contains([]byte(tracker.Message), []byte(fragment)) {
			matches = append(matches, tracker)
		}
	}
	return matches
}
