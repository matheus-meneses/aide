package webui

import (
	"aide/cli/internal/platform/clog"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

const (
	tailWindowBytes = 64 << 10
	tailMaxLines    = 500
	pollInterval    = 300 * time.Millisecond
)

func registerLogs(mux *http.ServeMux, path string) {
	mux.HandleFunc("GET /api/logs", func(w http.ResponseWriter, r *http.Request) {
		streamLogs(w, r, path)
	})
	mux.HandleFunc("DELETE /api/logs", func(w http.ResponseWriter, _ *http.Request) {
		if err := clog.ClearFile(); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
}

func streamLogs(w http.ResponseWriter, r *http.Request, path string) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	flusher.Flush()

	offset := emitBacklog(w, path)
	flusher.Flush()

	var leftover []byte
	poll := time.NewTicker(pollInterval)
	defer poll.Stop()
	heartbeat := time.NewTicker(15 * time.Second)
	defer heartbeat.Stop()

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case <-heartbeat.C:
			fmt.Fprint(w, ": keepalive\n\n")
			flusher.Flush()
		case <-poll.C:
			info, err := os.Stat(path)
			if err != nil {
				continue
			}
			size := info.Size()
			if size < offset {
				offset = 0
				leftover = leftover[:0]
			}
			if size <= offset {
				continue
			}
			data, newOffset, err := readRange(path, offset, size)
			if err != nil {
				continue
			}
			offset = newOffset
			leftover = append(leftover, data...)
			leftover = emitLines(w, leftover)
			flusher.Flush()
		}
	}
}

// emitBacklog seeds a freshly connected client with the tail of the existing
// file: the last tailMaxLines complete lines within a bounded window. It
// returns the file size, which becomes the follow offset.
func emitBacklog(w http.ResponseWriter, path string) int64 {
	info, err := os.Stat(path)
	if err != nil {
		return 0
	}
	size := info.Size()

	start := int64(0)
	if size > tailWindowBytes {
		start = size - tailWindowBytes
	}
	data, _, err := readRange(path, start, size)
	if err != nil {
		return size
	}
	if start > 0 {
		if i := bytes.IndexByte(data, '\n'); i >= 0 {
			data = data[i+1:]
		}
	}

	lines := bytes.Split(bytes.TrimRight(data, "\n"), []byte("\n"))
	if len(lines) > tailMaxLines {
		lines = lines[len(lines)-tailMaxLines:]
	}
	for _, line := range lines {
		writeLogLine(w, line)
	}
	return size
}

// emitLines writes every complete newline-terminated entry in buf and returns
// the trailing partial line for the next read.
func emitLines(w http.ResponseWriter, buf []byte) []byte {
	for {
		i := bytes.IndexByte(buf, '\n')
		if i < 0 {
			break
		}
		writeLogLine(w, buf[:i])
		buf = buf[i+1:]
	}
	rest := make([]byte, len(buf))
	copy(rest, buf)
	return rest
}

func writeLogLine(w http.ResponseWriter, line []byte) {
	line = bytes.TrimSpace(line)
	if len(line) == 0 {
		return
	}
	var entry clog.LogEntry
	if err := json.Unmarshal(line, &entry); err != nil {
		return
	}
	fmt.Fprintf(w, "data: %s\n\n", line)
}

func readRange(path string, start, end int64) ([]byte, int64, error) {
	if end <= start {
		return nil, start, nil
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, start, err
	}
	defer f.Close()

	buf := make([]byte, end-start)
	n, err := f.ReadAt(buf, start)
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, start, err
	}
	return buf[:n], start + int64(n), nil
}
