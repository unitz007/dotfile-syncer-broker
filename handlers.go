package main

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/r3labs/sse/v2"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	streamParam = "stream"
	blank       = ""
)

type SyncEvent struct {
	Data struct {
		Progress  int    `json:"progress"`
		IsSuccess bool   `json:"isSuccess"`
		Step      string `json:"step"`
		Error     string `json:"error"`
		Done      bool   `json:"done"`
	}
}

type Handlers struct {
	store            MachinesStore
	machineServer    *sse.Server
	syncEventServer  *sse.Server
	syncStatusServer *sse.Server
	webHookServer    *sse.Server
}

func NewHandler(
	store MachinesStore,
	syncStatusServer *sse.Server,
	syncEventServer *sse.Server,
	machineServer *sse.Server,
	webHookServer *sse.Server,
) *Handlers {

	syncEventServer.EventTTL = time.Second
	syncStatusServer.EventTTL = time.Second
	webHookServer.EventTTL = time.Second

	webHookServer.CreateStream("git-web-hook")

	for _, c := range *store.GetAll() {
		syncEventServer.CreateStream(c.Id)
		syncStatusServer.CreateStream(c.Id)
	}

	return &Handlers{
		syncEventServer:  syncEventServer,
		syncStatusServer: syncStatusServer,
		store:            store,
		machineServer:    machineServer,
		webHookServer:    webHookServer,
	}
}

func (m *Handlers) MachineHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		getMachines(w, r, m.store)
	case http.MethodPost:
		requestBody, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(err.Error()))
			return
		}

		var machine *Machine
		err = json.Unmarshal(requestBody, &machine)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(err.Error()))
			return
		}

		m.syncEventServer.CreateStream(machine.Id)
		m.syncStatusServer.CreateStream(machine.Id)

		err = m.store.Add(machine)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(err.Error()))
			return
		}

		w.WriteHeader(http.StatusNoContent)

	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (m *Handlers) SyncStatusHandler(w http.ResponseWriter, r *http.Request) {
	machineId := mux.Vars(r)["id"]
	if machineId == blank {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("Machine Id is required"))
		return
	}

	switch r.Method {
	case http.MethodPost:
		stream := m.syncStatusServer.CreateStream(machineId)
		if len(stream.Eventlog) != 0 {
			event := stream.Eventlog[len(stream.Eventlog)-1]
			stream.Eventlog = []*sse.Event{event}
		}

		defer func(body io.ReadCloser) {
			err := body.Close()
			if err != nil {
				fmt.Println(err)
			}
		}(r.Body)

		b, _ := io.ReadAll(r.Body)

		go func() {
			var syncStatus struct {
				LocalCommit struct {
					Id string `json:"id"`
				} `json:"local_commit"`
			}

			err := json.Unmarshal(b, &syncStatus)
			if err != nil {
				fmt.Println("could not update local commit for ", machineId, err.Error())
				return
			}

			machine, err := m.store.Get(machineId)
			if err != nil {
				fmt.Println("could not update local commit for ", machineId, err.Error())
				return
			}

			machine.SyncStatus.LocalCommit = syncStatus.LocalCommit.Id

			err = m.store.Add(machine)
			if err != nil {
				fmt.Println("could not update local commit for ", machineId, err.Error())
				return
			}
		}()

		m.syncStatusServer.Publish(machineId, &sse.Event{Data: b})
	case http.MethodGet:
		r.URL.RawQuery = func() string {
			q := r.URL.Query()
			q.Add(streamParam, machineId)
			return q.Encode()
		}()

		m.syncStatusServer.ServeHTTP(w, r)
		go func() {
			<-r.Context().Done()
			return
		}()
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (m *Handlers) SyncEventHandler(w http.ResponseWriter, r *http.Request) {
	machineId := mux.Vars(r)["id"]

	if machineId == blank {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("Machine Id is required"))
		return
	}

	_, err := m.store.Get(machineId)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(err.Error()))
		return
	}

	switch r.Method {
	case http.MethodPost:
		defer func(body io.ReadCloser) {
			err := body.Close()
			if err != nil {
				fmt.Println(err)
			}
		}(r.Body)

		exists := m.syncEventServer.StreamExists(machineId)
		if !exists {
			m.syncEventServer.CreateStream(machineId)
		}

		var event SyncEvent
		err = json.NewDecoder(r.Body).Decode(&event)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(err.Error()))
			return
		}

		a, err := json.Marshal(event.Data)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(err.Error()))
			return
		}
		m.syncEventServer.Publish(machineId, &sse.Event{Data: a})

		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(event.Data)

	case http.MethodGet:
		r.URL.RawQuery = func() string {
			q := r.URL.Query()
			q.Add(streamParam, machineId)
			return q.Encode()
		}()

		m.syncEventServer.ServeHTTP(w, r)
		go func() {
			<-r.Context().Done()
			return
		}()

	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (m *Handlers) WebHookHandler(w http.ResponseWriter, r *http.Request) {

	switch r.Method {
	case http.MethodPost:
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
		m.webHookServer.Publish("git-web-hook", &sse.Event{Data: []byte(c)})

		w.WriteHeader(http.StatusOK)

	case http.MethodGet:
		m.webHookServer.ServeHTTP(w, r)
		go func() {
			<-r.Context().Done()
			return
		}()

	}

}

func getMachines(w http.ResponseWriter, r *http.Request, store MachinesStore) {

	w.Header().Set("Content-Type", "application/json")

	machineId := mux.Vars(r)["id"]
	if machineId == blank {
		machines := store.GetAll()
		_ = json.NewEncoder(w).Encode(machines)
	} else {
		machine, err := store.Get(machineId)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte("Machine with id \"" + machineId + "\" not found"))
			return
		}

		_ = json.NewEncoder(w).Encode(machine)
	}
}
