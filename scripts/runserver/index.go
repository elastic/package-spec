package main

import (
	"net/http"

	"github.com/elastic/package-spec/v2/scripts/runserver/util"
)

type indexData struct {
	Service string
}

const serviceName = "Package Spec Debugger"

func indexHandler() (func(w http.ResponseWriter, r *http.Request), error) {
	data := indexData{
		Service: serviceName,
	}
	body, err := util.MarshalJSONPretty(&data)
	if err != nil {
		return nil, err
	}
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(body)
	}, nil
}
