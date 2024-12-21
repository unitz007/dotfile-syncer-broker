package handlers

import (
	"dotfile-syncer-broker/lib"
	"encoding/json"
	"github.com/gorilla/mux"
	"github.com/r3labs/sse/v2"
	"net/http"
)

type MachineHandler struct {
	Store             lib.MachinesStore
	MachineServer     *sse.Server
	SyncTriggerServer *sse.Server
	SyncStatusServer  *sse.Server
}

func (m *MachineHandler) GetMachines(w http.ResponseWriter, r *http.Request) {
	stream := r.URL.Query().Get("stream")

	if stream == "" {
		stores := m.Store.Get()
		w.Header().Set("Content-Type", "application/json")
		v, _ := json.Marshal(stores)
		_, _ = w.Write(v)
		return
	} else {
		m.MachineServer.ServeHTTP(w, r)
	}
}

func (m *MachineHandler) AddMachine(w http.ResponseWriter, r *http.Request) {
	machineId := mux.Vars(r)["machine-id"]

	m.SyncTriggerServer.CreateStream(machineId)
	m.SyncStatusServer.CreateStream(machineId)

	m.Store.Add(machineId)

	w.WriteHeader(http.StatusNoContent)
}
