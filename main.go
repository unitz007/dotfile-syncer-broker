package main

import (
	"fmt"
	"github.com/gorilla/mux"
	"github.com/r3labs/sse/v2"
	"log"
	"net/http"
)

const (
	MachinePath = "/machines"
)

func main() {

	router := mux.NewRouter()
	handler := NewHandler(
		NewStore(),
		sse.New(),
		sse.New(),
		sse.New(),
		sse.New(),
	)

	router.Methods(http.MethodGet, http.MethodPost).Path(MachinePath).HandlerFunc(handler.MachineHandler)
	router.Methods(http.MethodGet).Path(MachinePath + "/{id}").HandlerFunc(handler.MachineHandler)
	router.Methods(http.MethodGet, http.MethodPost).Path(MachinePath + "/{id}/sync-event").HandlerFunc(handler.SyncEventHandler)
	router.Methods(http.MethodGet, http.MethodPost).Path(MachinePath + "/{id}/sync-status").HandlerFunc(handler.SyncStatusHandler)
	router.Methods(http.MethodGet, http.MethodPost).Path("/git-hook").HandlerFunc(handler.WebHookHandler)

	mux.CORSMethodMiddleware(router)

	go func() {
		_ = router.Walk(func(route *mux.Route, router *mux.Router, ancestors []*mux.Route) error {
			s, _ := route.GetPathTemplate()
			methods, _ := route.GetMethods()
			fmt.Printf("==> %s %v\n", s, methods)
			return nil
		})
	}()

	fmt.Println("Listening on port 8080")

	log.Fatal(http.ListenAndServe(":8080", router))
}
