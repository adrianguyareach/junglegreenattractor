package event

import (
	"fmt"
	"sync"
	"time"
)

type Kind string

const (
	PipelineStarted   Kind = "pipeline.started"
	PipelineCompleted  Kind = "pipeline.completed"
	PipelineFailed     Kind = "pipeline.failed"
	StageStarted       Kind = "stage.started"
	StageCompleted     Kind = "stage.completed"
	StageFailed        Kind = "stage.failed"
	StageRetrying      Kind = "stage.retrying"
	InterviewStarted   Kind = "interview.started"
	InterviewCompleted Kind = "interview.completed"
	InterviewTimeout   Kind = "interview.timeout"
	CheckpointSaved    Kind = "checkpoint.saved"
	ParallelStarted    Kind = "parallel.started"
	ParallelCompleted  Kind = "parallel.completed"
)

type Event struct {
	Kind      Kind
	Timestamp time.Time
	NodeID    string
	Message   string
	Data      map[string]any
}

func (e Event) String() string {
	if e.NodeID != "" {
		return fmt.Sprintf("[%s] %s: %s", e.Timestamp.Format("15:04:05"), e.Kind, e.NodeID)
	}
	return fmt.Sprintf("[%s] %s: %s", e.Timestamp.Format("15:04:05"), e.Kind, e.Message)
}

type Handler func(Event)

type Emitter struct {
	mu       sync.RWMutex
	handlers []Handler
}

func NewEmitter() *Emitter {
	return &Emitter{}
}

func (e *Emitter) On(h Handler) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.handlers = append(e.handlers, h)
}

func (e *Emitter) Emit(evt Event) {
	if evt.Timestamp.IsZero() {
		evt.Timestamp = time.Now()
	}
	e.mu.RLock()
	defer e.mu.RUnlock()
	for _, h := range e.handlers {
		h(evt)
	}
}
