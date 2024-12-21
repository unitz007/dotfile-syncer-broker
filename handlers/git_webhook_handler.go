package handlers

import (
	"fmt"
	"github.com/r3labs/sse/v2"
	"io"
	"net/http"
	"strings"
)

type GitWebhookHandler struct {
	SseServer *sse.Server
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

	var c = strings.ReplaceAll(string(b), "\n", "")
	h.SseServer.Publish("git-web-hook", &sse.Event{Data: []byte(c)})

	w.WriteHeader(http.StatusOK)
}

func (h *GitWebhookHandler) Listen(w http.ResponseWriter, r *http.Request) {
	h.SseServer.ServeHTTP(w, r)
}
