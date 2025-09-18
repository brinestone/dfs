package api

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/brinestone/dfs/events"
)

const MaxSseMessageRetryCount = 100

type SseEvent struct {
	ID      []byte
	Data    []byte
	Event   []byte
	Retry   []byte
	Comment []byte
}

func (ev *SseEvent) MarshalTo(w io.Writer) error {
	if len(ev.Data) == 0 && len(ev.Comment) == 0 {
		return nil
	}

	if len(ev.Data) > 0 {
		if _, err := fmt.Fprintf(w, "id: %s\n", ev.ID); err != nil {
			return err
		}

		sd := bytes.Split(ev.Data, []byte("\n"))
		for i := range sd {
			if _, err := fmt.Fprintf(w, "data: %s\n", sd[i]); err != nil {
				return err
			}
		}

		if len(ev.Event) > 0 {
			if _, err := fmt.Fprintf(w, "event: %s\n", ev.Event); err != nil {
				return err
			}
		}

		if len(ev.Retry) > 0 {
			if _, err := fmt.Fprintf(w, "retry: %s\n", ev.Retry); err != nil {
				return err
			}
		}
	}
	if len(ev.Comment) > 0 {
		if _, err := fmt.Fprintf(w, ": %s\n", ev.Comment); err != nil {
			return err
		}
	}

	if _, err := fmt.Fprint(w, "\n"); err != nil {
		return err
	}

	return nil
}

func sseHandler(eventSource <-chan events.Event, logger *slog.Logger) http.Handler {
	logger = logger.WithGroup("sse")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := r.RemoteAddr[0:strings.LastIndex(r.RemoteAddr, ":")]
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		logger.Info("sse connection opened", "ip", ip)
		defer logger.Info("sse connection closed", "ip", ip)
		failureCount := 0
		var currentId uint64 = 0

		for {
			select {
			case <-r.Context().Done():
				return
			case data, ok := <-eventSource:
				if !ok {
					return
				}
				id := make([]byte, 8)
				binary.BigEndian.PutUint64(id, currentId)
				jsonData, err := json.Marshal(data)
				if err != nil {
					logger.Error("could not send message", "id", currentId, "ip", ip)
					failureCount = 0
					currentId++
					continue
				}
				event := SseEvent{
					ID:   id,
					Data: jsonData,
				}
				if failureCount > 0 {
					event.Retry = id
				}

				if err := event.MarshalTo(w); err != nil {
					logger.Error("error while writing sse message to response", "id", string(event.ID), "ip", ip, "err", err)
					failureCount++
					if failureCount > MaxSseMessageRetryCount {
						logger.Error("maximum sse message retry count reached, closing connection", "id", string(event.ID), "max", MaxSseMessageRetryCount, "ip", ip)
						return
					}
					logger.Warn("will retry sending sse message again in 10 seconds", "id", string(event.ID), "ip", ip, "currentCount", failureCount)
					time.Sleep(10 * time.Second)
					logger.Warn("retrying sse message", "id", string(event.ID), "ip", ip, "currentCount", failureCount)
					continue
				}
				currentId++
				failureCount = 0
			}
		}
	})
}
