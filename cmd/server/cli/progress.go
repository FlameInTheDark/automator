package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	prettyprogress "github.com/jedib0t/go-pretty/v6/progress"

	"github.com/FlameInTheDark/emerald/internal/pipeline"
)

type cliProgressWriter interface {
	AppendTracker(tracker *prettyprogress.Tracker)
	Render()
	SetAutoStop(autoStop bool)
	SetMessageLength(length int)
	SetOutputWriter(output io.Writer)
	SetPinnedMessages(messages ...string)
	SetSortBy(sortBy prettyprogress.SortBy)
	SetStyle(style prettyprogress.Style)
	SetTrackerLength(length int)
	SetUpdateFrequency(frequency time.Duration)
	Stop()
}

func newCLIProgressWriter(output io.Writer) cliProgressWriter {
	writer := prettyprogress.NewWriter()
	writer.SetOutputWriter(output)
	writer.SetAutoStop(false)
	writer.SetSortBy(prettyprogress.SortByIndex)
	style := prettyprogress.StyleDefault
	style.Visibility.TrackerOverall = false
	style.Visibility.Percentage = false
	style.Visibility.ETA = false
	style.Visibility.Time = true
	writer.SetStyle(style)
	writer.SetTrackerLength(14)
	writer.SetMessageLength(48)
	writer.SetUpdateFrequency(100 * time.Millisecond)
	return writer
}

type pipelineProgressAdapter struct {
	writer       cliProgressWriter
	pipelineName string
	nodeLabels   map[string]string

	mu          sync.Mutex
	trackers    map[string]*prettyprogress.Tracker
	finished    map[string]bool
	executionID string
	currentNode string
	doneCount   int
	failedCount int
	nextIndex   uint64
	renderDone  chan struct{}
	started     bool
	stopped     bool
}

func newPipelineProgressAdapter(pipelineName string, flowData pipeline.FlowData, writer cliProgressWriter) *pipelineProgressAdapter {
	return &pipelineProgressAdapter{
		writer:       writer,
		pipelineName: pipelineName,
		nodeLabels:   extractPipelineNodeLabels(flowData),
		trackers:     make(map[string]*prettyprogress.Tracker),
		finished:     make(map[string]bool),
		renderDone:   make(chan struct{}),
	}
}

func (a *pipelineProgressAdapter) Start() {
	if a == nil || a.writer == nil {
		return
	}

	a.mu.Lock()
	if a.started {
		a.mu.Unlock()
		return
	}
	a.started = true
	a.updatePinnedMessagesLocked()
	a.mu.Unlock()

	go func() {
		a.writer.Render()
		close(a.renderDone)
	}()
}

func (a *pipelineProgressAdapter) Stop() {
	if a == nil || a.writer == nil {
		return
	}

	a.mu.Lock()
	if !a.started || a.stopped {
		a.mu.Unlock()
		return
	}
	a.stopped = true
	a.mu.Unlock()

	a.writer.Stop()
	<-a.renderDone
}

func (a *pipelineProgressAdapter) HandleEvent(event pipeline.ProgressEvent) {
	if a == nil || a.writer == nil {
		return
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	switch event.Kind {
	case pipeline.ProgressEventExecutionStarted:
		a.executionID = event.ExecutionID
	case pipeline.ProgressEventNodeStarted:
		a.currentNode = a.nodeDisplay(event.NodeID, event.NodeType)
		if _, exists := a.trackers[event.NodeID]; !exists {
			a.trackers[event.NodeID] = &prettyprogress.Tracker{
				AutoStopDisabled: true,
				Index:            a.nextIndex,
				Message:          a.currentNode,
			}
			a.nextIndex++
			a.writer.AppendTracker(a.trackers[event.NodeID])
		}
	case pipeline.ProgressEventNodeCompleted:
		if tracker, exists := a.trackers[event.NodeID]; exists {
			if event.Status == "failed" {
				tracker.MarkAsErrored()
			} else {
				tracker.MarkAsDone()
			}
		}
		if !a.finished[event.NodeID] {
			a.finished[event.NodeID] = true
			if event.Status == "failed" {
				a.failedCount++
			} else {
				a.doneCount++
			}
		}
		if strings.EqualFold(a.currentNode, a.nodeDisplay(event.NodeID, event.NodeType)) {
			a.currentNode = ""
		}
	case pipeline.ProgressEventExecutionCompleted:
		a.currentNode = ""
	}

	a.updatePinnedMessagesLocked()
}

func (a *pipelineProgressAdapter) updatePinnedMessagesLocked() {
	currentNode := a.currentNode
	if strings.TrimSpace(currentNode) == "" {
		currentNode = "waiting"
	}

	executionID := a.executionID
	if strings.TrimSpace(executionID) == "" {
		executionID = "pending"
	}

	a.writer.SetPinnedMessages(
		fmt.Sprintf("Pipeline: %s", a.pipelineName),
		fmt.Sprintf("Execution ID: %s", executionID),
		fmt.Sprintf("Current Node: %s", currentNode),
		fmt.Sprintf("Done/Failed: %d/%d", a.doneCount, a.failedCount),
	)
}

func (a *pipelineProgressAdapter) nodeDisplay(nodeID string, nodeType string) string {
	label := strings.TrimSpace(a.nodeLabels[nodeID])
	if label == "" {
		label = nodeID
	}
	return fmt.Sprintf("%s (%s)", label, nodeType)
}

func extractPipelineNodeLabels(flowData pipeline.FlowData) map[string]string {
	labels := make(map[string]string, len(flowData.Nodes))
	for _, flowNode := range flowData.Nodes {
		if label := extractNodeLabel(flowNode.Data); label != "" {
			labels[flowNode.ID] = label
		}
	}
	return labels
}

func extractNodeLabel(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}

	var payload map[string]json.RawMessage
	if err := json.Unmarshal(raw, &payload); err != nil {
		return ""
	}

	labelRaw, ok := payload["label"]
	if !ok {
		return ""
	}

	var label string
	if err := json.Unmarshal(labelRaw, &label); err != nil {
		return ""
	}

	return strings.TrimSpace(label)
}
