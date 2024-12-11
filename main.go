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

	router := mux.NewRouter()

	store := NewStore()

	// register machines
	for _, c := range store.Get() {
		syncTriggerServer.CreateStream(c)
		syncStatusServer.CreateStream(c)
	}

	// handlers
	syncTriggerHandler := SyncTriggerHandler{
		server: syncTriggerServer,
		store:  store,
	}
	syncStatusHandler := SyncStatusHandler{
		server: syncStatusServer,
		store:  store,
	}

	// sync trigger handlers
	router.Methods("POST").Path("/sync-trigger/{machine-id}/notify").HandlerFunc(syncTriggerHandler.SyncNotify)
	router.Methods("GET").Path("/sync-trigger").HandlerFunc(syncTriggerHandler.Status)

	// sync trigger handlers
	router.Methods("POST").Path("/sync-status/{machine-id}/notify").HandlerFunc(syncStatusHandler.SyncStatusNotify)
	router.Methods("GET").Path("/sync-status").HandlerFunc(syncStatusHandler.SyncStatus)

	router.Path("/machines").Methods("GET").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		machineServer.ServeHTTP(w, r)
	})

	router.Path("/machines/{machine-id}").Methods("POST").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		machineId := mux.Vars(r)["machine-id"]
		syncTriggerServer.CreateStream(machineId)
		syncStatusServer.CreateStream(machineId)

		store.Add(machineId)

		w.WriteHeader(http.StatusNoContent)
	})

	c := cors.Default().Handler(router)
	log.Fatal(http.ListenAndServe(":8080", c))
}
