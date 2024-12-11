package main

import (
	"github.com/gorilla/mux"
	"github.com/r3labs/sse/v2"
	"github.com/rs/cors"
	"log"
	"net/http"
	"time"
)

func main() {

	machineServer := sse.New()
	machineServer.CreateStream("machine")

	syncTriggerServer := sse.New()
	syncStatusServer := sse.New()

	syncTriggerServer.EventTTL = time.Second

	m := mux.NewRouter()

	ms := NewStore(machineServer)

	// register streams
	for _, c := range ms.Get() {
		syncTriggerServer.CreateStream(c)
		syncStatusServer.CreateStream(c)
	}

	syncTriggerHandler := SyncTriggerHandler{
		Server: syncTriggerServer,
		M:      ms,
	}
	syncStatusHandler := SyncStatusHandler{
		Server: syncStatusServer,
		M:      ms,
	}

	// sync trigger handlers
	m.Methods("POST").Path("/sync-trigger/{machine-id}/notify").HandlerFunc(syncTriggerHandler.SyncNotify)
	m.Methods("GET").Path("/sync-trigger").HandlerFunc(syncTriggerHandler.Status)

	// sync trigger handlers
	m.Methods("POST").Path("/sync-status/{machine-id}/notify").HandlerFunc(syncStatusHandler.SyncStatusNotify)
	m.Methods("GET").Path("/sync-status").HandlerFunc(syncStatusHandler.SyncStatus)

	m.Path("/machines").Methods("GET").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		machineServer.ServeHTTP(w, r)
	})

	m.Path("/machines/{machine-id}").Methods("POST").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		machineId := mux.Vars(r)["machine-id"]
		syncTriggerServer.CreateStream(machineId)
		syncStatusServer.CreateStream(machineId)

		ms.Add(machineId)

		w.WriteHeader(http.StatusNoContent)
	})

	c := cors.Default().Handler(m)
	log.Fatal(http.ListenAndServe(":8080", c))
}
