// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !wasm

package main

import (
	"log"
	"net/http"
)

func main() {
	staticHandler := http.FileServer(http.Dir("."))
	http.Handle("/", staticHandler)
	log.Fatal(http.ListenAndServe("localhost:8080", nil))
}
