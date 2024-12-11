package main

import (
	"fmt"
	"github.com/gorilla/mux"
	"github.com/r3labs/sse/v2"
	"io"
	"net/http"
)

type SyncStatusHandler struct {
	server *sse.Server
	store  MachinesStore
}

func (h *SyncStatusHandler) SyncStatusNotify(w http.ResponseWriter, r *http.Request) {
	defer func(body io.ReadCloser) {
		err := body.Close()
		if err != nil {
			fmt.Println(err)
		}
	}(r.Body)

	machineId := mux.Vars(r)["machine-id"]
	if machineId == "" {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("Machine Id is required"))
		return
	}

	stream := h.server.CreateStream(machineId)
	if len(stream.Eventlog) != 0 {
		event := stream.Eventlog[len(stream.Eventlog)-1]
		stream.Eventlog = []*sse.Event{event}
	}

	h.store.Add(machineId)

	b, _ := io.ReadAll(r.Body)

	h.server.Publish(machineId, &sse.Event{Data: b})
}

func (h *SyncStatusHandler) SyncStatus(w http.ResponseWriter, r *http.Request) {

	h.server.ServeHTTP(w, r)
	go func() {
		<-r.Context().Done()
		return
	}()
}
