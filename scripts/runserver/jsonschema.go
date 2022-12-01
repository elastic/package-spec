package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/Masterminds/semver/v3"
	"github.com/gorilla/mux"

	"github.com/elastic/package-spec/v2/code/go/pkg/jsonschema"
)

const jsonschemaRouterPath = "/jsonschema/{packageType}/{version}/{filepath}"

func jsonschemaHandler() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		packageType, ok := vars["packageType"]
		if !ok {
			badRequest(w, "missing package type")
			return
		}

		specVersion, ok := vars["version"]
		if !ok {
			badRequest(w, "missing version package spec")
			return
		}

		filepath, ok := vars["filepath"]
		if !ok {
			badRequest(w, "missing file path to check")
			return
		}

		_, err := semver.StrictNewVersion(specVersion)
		if err != nil {
			badRequest(w, "invalid package spec version")
			return
		}

		serveJsonSchema(w, r, packageType, specVersion, filepath)
	}
}

func serveJsonSchema(w http.ResponseWriter, r *http.Request, packageType, specVersion, filepath string) {
	rendered, err := jsonschema.JSONSchema(filepath, specVersion, packageType)
	if err != nil {
		log.Printf("Error %s", err)
		internalServerError(w, "not able to render the json schema")
		return
	}

	yamlHeader(w)
	fmt.Fprint(w, string(rendered))
}
