package worktree

import (
	"fmt"
	"io"
	"sync"
	"time"
)

const sweepProgressInterval = 90 * time.Millisecond

var sweepProgressFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

type sweepProgress struct {
	writer  io.Writer
	enabled bool

	mu      sync.Mutex
	writeMu sync.Mutex
	message string
	running bool
	stop    chan struct{}
	done    chan struct{}
}

func newSweepProgress(writer io.Writer, enabled bool) *sweepProgress {
	return &sweepProgress{writer: writer, enabled: enabled && writer != nil}
}

func (progress *sweepProgress) start(message string) {
	if progress == nil || !progress.enabled {
		return
	}
	progress.stopLine()
	progress.mu.Lock()
	progress.message = sanitizeTerminalField(message)
	progress.running = true
	progress.stop = make(chan struct{})
	progress.done = make(chan struct{})
	stop, done := progress.stop, progress.done
	progress.mu.Unlock()
	progress.writeFrame(0)
	go func() {
		defer close(done)
		ticker := time.NewTicker(sweepProgressInterval)
		defer ticker.Stop()
		frame := 1
		for {
			select {
			case <-ticker.C:
				progress.writeFrame(frame)
				frame = (frame + 1) % len(sweepProgressFrames)
			case <-stop:
				return
			}
		}
	}()
}

func (progress *sweepProgress) update(format string, values ...any) {
	if progress == nil || !progress.enabled {
		return
	}
	progress.mu.Lock()
	progress.message = sanitizeTerminalField(fmt.Sprintf(format, values...))
	progress.mu.Unlock()
	progress.writeFrame(0)
}

func (progress *sweepProgress) stopLine() {
	if progress == nil || !progress.enabled {
		return
	}
	progress.mu.Lock()
	if !progress.running {
		progress.mu.Unlock()
		return
	}
	progress.running = false
	stop, done := progress.stop, progress.done
	close(stop)
	progress.mu.Unlock()
	<-done
	_, _ = fmt.Fprint(progress.writer, "\r\x1b[2K")
}

func (progress *sweepProgress) writeFrame(frame int) {
	progress.mu.Lock()
	message, running := progress.message, progress.running
	progress.mu.Unlock()
	if !running {
		return
	}
	progress.writeMu.Lock()
	defer progress.writeMu.Unlock()
	_, _ = fmt.Fprintf(progress.writer, "\r\x1b[2K%s %s", sweepProgressFrames[frame], message)
}
