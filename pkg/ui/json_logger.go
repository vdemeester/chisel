package ui

import (
	"encoding/json"
	"io"
	"sync"
	"time"
)

// JSONLogger outputs structured JSON lines.
type JSONLogger struct {
	w  io.Writer
	mu *sync.Mutex
}

// LogEvent represents a structured log event.
type LogEvent struct {
	Time     string         `json:"time"`
	Type     string         `json:"type"`
	Name     string         `json:"name,omitempty"`
	Image    string         `json:"image,omitempty"`
	Duration string         `json:"duration,omitempty"`
	Status   string         `json:"status,omitempty"`
	Level    string         `json:"level,omitempty"`
	Message  string         `json:"message,omitempty"`
	Output   string         `json:"output,omitempty"`
	Error    string         `json:"error,omitempty"`
	Attrs    map[string]any `json:"attrs,omitempty"`
}

func (l *JSONLogger) emit(event LogEvent) {
	event.Time = time.Now().UTC().Format(time.RFC3339Nano)
	l.mu.Lock()
	defer l.mu.Unlock()
	data, _ := json.Marshal(event)
	_, _ = l.w.Write(data)
	_, _ = l.w.Write([]byte("\n"))
}

func (l *JSONLogger) PipelineStart(name string) {
	l.emit(LogEvent{
		Type:   "pipeline_start",
		Name:   name,
		Status: "running",
	})
}

func (l *JSONLogger) PipelineEnd(name string, duration time.Duration, err error) {
	event := LogEvent{
		Type:     "pipeline_end",
		Name:     name,
		Duration: formatDuration(duration),
	}
	if err != nil {
		event.Status = "failed"
		event.Error = err.Error()
	} else {
		event.Status = "success"
	}
	l.emit(event)
}

func (l *JSONLogger) TaskStart(name string) {
	l.emit(LogEvent{
		Type:   "task_start",
		Name:   name,
		Status: "running",
	})
}

func (l *JSONLogger) TaskEnd(name string, duration time.Duration, err error) {
	event := LogEvent{
		Type:     "task_end",
		Name:     name,
		Duration: formatDuration(duration),
	}
	if err != nil {
		event.Status = "failed"
		event.Error = err.Error()
	} else {
		event.Status = "success"
	}
	l.emit(event)
}

func (l *JSONLogger) StepStart(name string, image string) {
	l.emit(LogEvent{
		Type:   "step_start",
		Name:   name,
		Image:  image,
		Status: "running",
	})
}

func (l *JSONLogger) StepEnd(name string, duration time.Duration, err error) {
	event := LogEvent{
		Type:     "step_end",
		Name:     name,
		Duration: formatDuration(duration),
	}
	if err != nil {
		event.Status = "failed"
		event.Error = err.Error()
	} else {
		event.Status = "success"
	}
	l.emit(event)
}

func (l *JSONLogger) StepOutput(name string, output string) {
	if output == "" {
		return
	}
	l.emit(LogEvent{
		Type:   "step_output",
		Name:   name,
		Output: output,
	})
}

func (l *JSONLogger) Debug(msg string, attrs ...any) {
	l.emit(LogEvent{
		Type:    "log",
		Level:   "debug",
		Message: msg,
		Attrs:   attrsToMap(attrs),
	})
}

func (l *JSONLogger) Info(msg string, attrs ...any) {
	l.emit(LogEvent{
		Type:    "log",
		Level:   "info",
		Message: msg,
		Attrs:   attrsToMap(attrs),
	})
}

func (l *JSONLogger) Warn(msg string, attrs ...any) {
	l.emit(LogEvent{
		Type:    "log",
		Level:   "warn",
		Message: msg,
		Attrs:   attrsToMap(attrs),
	})
}

func (l *JSONLogger) Error(msg string, attrs ...any) {
	l.emit(LogEvent{
		Type:    "log",
		Level:   "error",
		Message: msg,
		Attrs:   attrsToMap(attrs),
	})
}

func attrsToMap(attrs []any) map[string]any {
	if len(attrs) == 0 {
		return nil
	}
	m := make(map[string]any)
	for i := 0; i+1 < len(attrs); i += 2 {
		key, ok := attrs[i].(string)
		if !ok {
			continue
		}
		m[key] = attrs[i+1]
	}
	return m
}
