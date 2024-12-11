package handlers

import (
	"fmt"
	"github.com/r3labs/sse/v2"
	"io"
	"net/http"
)

type GitWebhookHandler struct {
	SseServer *sse.Server
	Stream    *sse.Stream
}

func (h *GitWebhookHandler) ReceivePushEvent(w http.ResponseWriter, r *http.Request) {

	h.Stream.Eventlog.Clear()

	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			fmt.Println(err)
		}
	}(r.Body)

	b, err := io.ReadAll(r.Body)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	h.SseServer.Publish("git-web-hook", &sse.Event{Data: b})

	w.WriteHeader(http.StatusOK)
}

func (h *GitWebhookHandler) Listen(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	//for x := range h.Events {
	//	_, _ = fmt.Fprintf(w, "data: %v\n\n", x)
	//	w.(http.Flusher).Flush() // Send the event immediately
	//}
	//
	//h

	h.SseServer.ServeHTTP(w, r)
}
