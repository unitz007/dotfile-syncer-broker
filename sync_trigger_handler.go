package main

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/r3labs/sse/v2"
	"io"
	"net/http"
)

type SyncTriggerHandler struct {
	Server *sse.Server
	M      MachinesStore
}

type SyncEvent struct {
	Data struct {
		Progress  int    `json:"progress"`
		IsSuccess bool   `json:"isSuccess"`
		Step      string `json:"step"`
		Error     string `json:"error"`
		Done      bool   `json:"done"`
	}
}

func (s *SyncTriggerHandler) SyncNotify(w http.ResponseWriter, r *http.Request) {
	machineId := mux.Vars(r)["machine-id"]
	if machineId == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Machine Id is required"))
		return
	}

	defer func(body io.ReadCloser) {
		err := body.Close()
		if err != nil {
			fmt.Println(err)
		}
	}(r.Body)

	exists := s.Server.StreamExists(machineId)
	if !exists {
		s.Server.CreateStream(machineId)
	}

	s.M.Add(machineId)
	var event SyncEvent
	_ = json.NewDecoder(r.Body).Decode(&event)

	a, _ := json.Marshal(event.Data)
	s.Server.Publish(machineId, &sse.Event{Data: a})

	w.WriteHeader(http.StatusOK)
}

func (s *SyncTriggerHandler) Status(w http.ResponseWriter, r *http.Request) {
	s.Server.ServeHTTP(w, r)
	go func() {
		<-r.Context().Done()
		return
	}()
}
