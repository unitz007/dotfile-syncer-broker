package main

import (
	"dotfile-syncer-broker/handlers"
	"dotfile-syncer-broker/lib"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/r3labs/sse/v2"
	"github.com/rs/cors"
)

func main() {

	machineServer := sse.New()
	machineServer.CreateStream("machine")

	syncTriggerServer := sse.New()
	syncStatusServer := sse.New()
	gitWebHookStreamServer := sse.New()
	gitWebHookStreamServer.EventTTL = time.Second

	syncTriggerServer.EventTTL = time.Second

	router := mux.NewRouter()

	store := lib.NewStore()

	// register machines
	for _, c := range store.Get() {
		syncTriggerServer.CreateStream(c)
		syncStatusServer.CreateStream(c)
	}

	// handlers
	syncTriggerHandler := handlers.SyncTriggerHandler{
		Server: syncTriggerServer,
		Store:  store,
	}
	syncStatusHandler := handlers.SyncStatusHandler{
		Server: syncStatusServer,
		Store:  store,
	}

	machineHandler := handlers.MachineHandler{
		Store:             store,
		MachineServer:     machineServer,
		SyncTriggerServer: syncTriggerServer,
		SyncStatusServer:  syncStatusServer,
	}

	gitWebHookStreamServer.CreateStream("git-web-hook")
	gitWebHookHandler := handlers.GitWebhookHandler{
		SseServer: gitWebHookStreamServer,
	}

	go func() {
		// keep alive
		for {
			gitWebHookStreamServer.Publish("git-web-hook", &sse.Event{Data: []byte("{}")})
			time.Sleep(2 * time.Second)
		}
	}()

	// sync handlers
	router.Methods("POST").Path("/sync-trigger/{machine-id}/notify").HandlerFunc(syncTriggerHandler.SyncNotify)
	router.Methods("GET").Path("/sync-trigger").HandlerFunc(syncTriggerHandler.Status)
	router.Methods("POST").Path("/sync-status/{machine-id}/notify").HandlerFunc(syncStatusHandler.SyncStatusNotify)
	router.Methods("GET").Path("/sync-status").HandlerFunc(syncStatusHandler.SyncStatus)
	router.Path("/machines").Methods("GET").HandlerFunc(machineHandler.GetMachines)
	router.Path("/machines/{machine-id}").Methods("POST").HandlerFunc(machineHandler.AddMachine)
	router.Methods("POST").Path("/git-hook").HandlerFunc(gitWebHookHandler.ReceivePushEvent)
	router.Methods("GET").Path("/git-hook").HandlerFunc(gitWebHookHandler.Listen)

	c := cors.Default().Handler(router)
	log.Fatal(http.ListenAndServe(":8080", c))
}
