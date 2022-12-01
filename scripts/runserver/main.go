package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/gorilla/mux"
)

var (
	address         string
	httpProfAddress string
)

func main() {

	parameters()

	flag.Parse()

	initHttpProf()

	log.Printf("Running debug server %s", address)
	server := initServer()
	go func() {
		err := runServer(server)
		if err != nil && err != http.ErrServerClosed {
			log.Fatal("error occurred while serving: %s", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	ctx := context.Background()
	if err := server.Shutdown(ctx); err != nil {
		log.Fatal("error on shutdown: %s", err)
	}
}

func runServer(server *http.Server) error {
	return server.ListenAndServe()
}

func initHttpProf() {
	if httpProfAddress == "" {
		return
	}

	log.Printf("Starting http pprof in %s", httpProfAddress)
	go func() {
		err := http.ListenAndServe(httpProfAddress, nil)
		if err != nil {
			log.Fatal("failed to start HTTP profiler: %s", err)
		}
	}()
}

func initServer() *http.Server {
	router, err := getRouter()
	if err != nil {
		log.Fatal("failed go configure router %s", err)
	}
	return &http.Server{Addr: address, Handler: router}
}

func getRouter() (*mux.Router, error) {

	indexHandlerFunc, err := indexHandler()
	if err != nil {
		return nil, err
	}

	jsonSchemaHandlerFunc := jsonschemaHandler()

	router := mux.NewRouter().StrictSlash(true)
	router.HandleFunc("/", indexHandlerFunc)
	router.HandleFunc(jsonschemaRouterPath, jsonSchemaHandlerFunc)
	router.NotFoundHandler = notFoundHandler(fmt.Errorf("404 page not found"))

	return router, nil
}

func notFoundHandler(err error) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		notFoundError(w, err)
	})
}

func notFoundError(w http.ResponseWriter, err error) {
	http.Error(w, err.Error(), http.StatusNotFound)
}

func badRequest(w http.ResponseWriter, errorMessage string) {
	http.Error(w, errorMessage, http.StatusBadRequest)
}

func internalServerError(w http.ResponseWriter, errorMessage string) {
	http.Error(w, errorMessage, http.StatusInternalServerError)
}

func jsonHeader(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
}

func yamlHeader(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/yaml")
}
