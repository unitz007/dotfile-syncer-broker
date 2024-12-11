package handlers

import (
	"fmt"
	"github.com/r3labs/sse/v2"
	"io"
	"net/http"
)

type GitWebhookHandler struct {
	SseServer *sse.Server
	Events    chan string
}

func (h *GitWebhookHandler) ReceivePushEvent(w http.ResponseWriter, r *http.Request) {

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

	h.Events <- string(b)

	w.WriteHeader(http.StatusOK)
}

func (h *GitWebhookHandler) Listen(w http.ResponseWriter, _ *http.Request) {

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	for x := range h.Events {
		_, _ = fmt.Fprintf(w, "data: %v\n\n", x)
		w.(http.Flusher).Flush() // Send the event immediately
	}
}
