package ui

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"
)

// Logger provides structured logging for pipeline execution.
type Logger interface {
	// Lifecycle events
	PipelineStart(name string)
	PipelineEnd(name string, duration time.Duration, err error)
	TaskStart(name string)
	TaskEnd(name string, duration time.Duration, err error)
	StepStart(name string, image string)
	StepEnd(name string, duration time.Duration, err error)

	// General logging
	Debug(msg string, attrs ...any)
	Info(msg string, attrs ...any)
	Warn(msg string, attrs ...any)
	Error(msg string, attrs ...any)

	// Output capture
	StepOutput(name string, output string)
}

// NewLogger creates a logger for the given output mode.
func NewLogger(mode OutputMode, w io.Writer) Logger {
	if w == nil {
		w = os.Stdout
	}
	switch mode {
	case OutputJSON:
		return &JSONLogger{w: w, mu: &sync.Mutex{}}
	case OutputPlain:
		return &PlainLogger{w: w, mu: &sync.Mutex{}}
	default:
		return &PrettyLogger{w: w, mu: &sync.Mutex{}}
	}
}

// PrettyLogger outputs colored, hierarchical logs.
type PrettyLogger struct {
	w     io.Writer
	mu    *sync.Mutex
	depth int
}

func (l *PrettyLogger) indent() string {
	return strings.Repeat("  ", l.depth)
}

func (l *PrettyLogger) write(format string, args ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	_, _ = fmt.Fprintf(l.w, format, args...)
}

func (l *PrettyLogger) PipelineStart(name string) {
	symbol := StyleRunning.Render(SymbolRunning)
	label := StyleLabel.Render("Pipeline:")
	l.write("%s%s %s %s\n", l.indent(), symbol, label, name)
	l.depth++
}

func (l *PrettyLogger) PipelineEnd(name string, duration time.Duration, err error) {
	l.depth--
	var symbol, status string
	if err != nil {
		symbol = StyleFailure.Render(SymbolFailure)
		status = StyleFailure.Render("Pipeline failed")
	} else {
		symbol = StyleSuccess.Render(SymbolSuccess)
		status = StyleSuccess.Render("Pipeline completed")
	}
	dur := StyleDuration.Render(fmt.Sprintf("[%s]", formatDuration(duration)))
	l.write("%s%s %s %s\n", l.indent(), symbol, status, dur)
}

func (l *PrettyLogger) TaskStart(name string) {
	symbol := StyleRunning.Render(SymbolRunning)
	label := StyleLabel.Render("Task:")
	l.write("%s%s %s %s\n", l.indent(), symbol, label, name)
	l.depth++
}

func (l *PrettyLogger) TaskEnd(name string, duration time.Duration, err error) {
	l.depth--
	var symbol string
	if err != nil {
		symbol = StyleFailure.Render(SymbolFailure)
	} else {
		symbol = StyleSuccess.Render(SymbolSuccess)
	}
	label := StyleLabel.Render("Task:")
	dur := StyleDuration.Render(fmt.Sprintf("[%s]", formatDuration(duration)))
	if err != nil {
		errMsg := StyleError.Render("ERROR")
		l.write("%s%s %s %s %s %s\n", l.indent(), symbol, label, name, dur, errMsg)
	} else {
		l.write("%s%s %s %s %s\n", l.indent(), symbol, label, name, dur)
	}
}

func (l *PrettyLogger) StepStart(name string, image string) {
	symbol := StyleRunning.Render(SymbolRunning)
	label := StyleLabel.Render("Step:")
	imageInfo := StyleDim.Render(fmt.Sprintf("(%s)", image))
	l.write("%s%s %s %s %s\n", l.indent(), symbol, label, name, imageInfo)
	l.depth++
}

func (l *PrettyLogger) StepEnd(name string, duration time.Duration, err error) {
	l.depth--
	var symbol string
	if err != nil {
		symbol = StyleFailure.Render(SymbolFailure)
	} else {
		symbol = StyleSuccess.Render(SymbolSuccess)
	}
	label := StyleLabel.Render("Step:")
	dur := StyleDuration.Render(fmt.Sprintf("[%s]", formatDuration(duration)))
	if err != nil {
		errMsg := StyleError.Render("ERROR")
		l.write("%s%s %s %s %s %s\n", l.indent(), symbol, label, name, dur, errMsg)
	} else {
		l.write("%s%s %s %s %s\n", l.indent(), symbol, label, name, dur)
	}
}

func (l *PrettyLogger) StepOutput(name string, output string) {
	if output == "" {
		return
	}
	lines := strings.Split(strings.TrimSpace(output), "\n")
	for _, line := range lines {
		arrow := StyleDim.Render(SymbolArrow)
		l.write("%s  %s %s\n", l.indent(), arrow, line)
	}
}

func (l *PrettyLogger) Debug(msg string, attrs ...any) {
	formatted := formatAttrs(msg, attrs...)
	l.write("%s%s\n", l.indent(), StyleDebug.Render(formatted))
}

func (l *PrettyLogger) Info(msg string, attrs ...any) {
	formatted := formatAttrs(msg, attrs...)
	l.write("%s%s\n", l.indent(), StyleInfo.Render(formatted))
}

func (l *PrettyLogger) Warn(msg string, attrs ...any) {
	formatted := formatAttrs(msg, attrs...)
	l.write("%s%s\n", l.indent(), StyleWarn.Render(formatted))
}

func (l *PrettyLogger) Error(msg string, attrs ...any) {
	formatted := formatAttrs(msg, attrs...)
	l.write("%s%s\n", l.indent(), StyleError.Render(formatted))
}

// PlainLogger outputs simple text without colors.
type PlainLogger struct {
	w     io.Writer
	mu    *sync.Mutex
	depth int
}

func (l *PlainLogger) indent() string {
	return strings.Repeat("  ", l.depth)
}

func (l *PlainLogger) write(format string, args ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	_, _ = fmt.Fprintf(l.w, format, args...)
}

func (l *PlainLogger) PipelineStart(name string) {
	l.write("%s* Pipeline: %s\n", l.indent(), name)
	l.depth++
}

func (l *PlainLogger) PipelineEnd(name string, duration time.Duration, err error) {
	l.depth--
	if err != nil {
		l.write("%sX Pipeline failed [%s]\n", l.indent(), formatDuration(duration))
	} else {
		l.write("%s+ Pipeline completed [%s]\n", l.indent(), formatDuration(duration))
	}
}

func (l *PlainLogger) TaskStart(name string) {
	l.write("%s* Task: %s\n", l.indent(), name)
	l.depth++
}

func (l *PlainLogger) TaskEnd(name string, duration time.Duration, err error) {
	l.depth--
	if err != nil {
		l.write("%sX Task: %s [%s] ERROR\n", l.indent(), name, formatDuration(duration))
	} else {
		l.write("%s+ Task: %s [%s]\n", l.indent(), name, formatDuration(duration))
	}
}

func (l *PlainLogger) StepStart(name string, image string) {
	l.write("%s* Step: %s (%s)\n", l.indent(), name, image)
	l.depth++
}

func (l *PlainLogger) StepEnd(name string, duration time.Duration, err error) {
	l.depth--
	if err != nil {
		l.write("%sX Step: %s [%s] ERROR\n", l.indent(), name, formatDuration(duration))
	} else {
		l.write("%s+ Step: %s [%s]\n", l.indent(), name, formatDuration(duration))
	}
}

func (l *PlainLogger) StepOutput(name string, output string) {
	if output == "" {
		return
	}
	lines := strings.Split(strings.TrimSpace(output), "\n")
	for _, line := range lines {
		l.write("%s  > %s\n", l.indent(), line)
	}
}

func (l *PlainLogger) Debug(msg string, attrs ...any) {
	formatted := formatAttrs(msg, attrs...)
	l.write("%s[DEBUG] %s\n", l.indent(), formatted)
}

func (l *PlainLogger) Info(msg string, attrs ...any) {
	formatted := formatAttrs(msg, attrs...)
	l.write("%s[INFO] %s\n", l.indent(), formatted)
}

func (l *PlainLogger) Warn(msg string, attrs ...any) {
	formatted := formatAttrs(msg, attrs...)
	l.write("%s[WARN] %s\n", l.indent(), formatted)
}

func (l *PlainLogger) Error(msg string, attrs ...any) {
	formatted := formatAttrs(msg, attrs...)
	l.write("%s[ERROR] %s\n", l.indent(), formatted)
}

// Helper functions

func formatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	return fmt.Sprintf("%.1fm", d.Minutes())
}

func formatAttrs(msg string, attrs ...any) string {
	if len(attrs) == 0 {
		return msg
	}
	var parts []string
	parts = append(parts, msg)
	for i := 0; i+1 < len(attrs); i += 2 {
		key := fmt.Sprintf("%v", attrs[i])
		value := fmt.Sprintf("%v", attrs[i+1])
		parts = append(parts, fmt.Sprintf("%s=%s", key, value))
	}
	return strings.Join(parts, " ")
}
