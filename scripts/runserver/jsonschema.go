package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/gorilla/mux"
	"gopkg.in/yaml.v3"

	"github.com/elastic/package-spec/v2/code/go/pkg/jsonschema"
)

const jsonschemaRouterPath = "/jsonschema/{packageType}/{version}/{filepath:.*}"

var supportedFormatsSet = map[string]string{
	"json": "json",
	"yaml": "yaml",
	"yml":  "yaml",
}

var supportedFormats = supportedFormatsList()

func supportedFormatsList() []string {
	keys := make([]string, 0, len(supportedFormatsSet))
	for k, _ := range supportedFormatsSet {
		keys = append(keys, k)
	}
	return keys
}

func jsonschemaHandler() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		outputFormat := "json"

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

		query := r.URL.Query()
		if v := query.Get("format"); v != "" {
			format := strings.ToLower(v)
			if _, ok := supportedFormatsSet[format]; !ok {
				message := fmt.Sprintf("invalid output format (%s). Options: %s", v, strings.Join(supportedFormats, ", "))
				badRequest(w, message)
				return
			}
			outputFormat = supportedFormatsSet[format]
		}

		serveJsonSchema(w, r, packageType, specVersion, filepath, outputFormat)
	}
}

func serveJsonSchema(w http.ResponseWriter, r *http.Request, packageType, specVersion, filepath, format string) {
	rendered, err := jsonschema.JSONSchema(filepath, specVersion, packageType)
	if err != nil {
		message := fmt.Sprintf("error: %s", err)
		log.Print(message)
		internalServerError(w, message)
		return
	}

	jsonSchemaBytes := len(rendered)
	if string(rendered) == "" {
		log.Printf("Empty jsonschema for this file: %s", filepath)
		jsonSchemaBytes = 0
	}

	switch format {
	case "json":
		log.Printf("Rendering jsonschema in json")
		var jsonSchema map[string]interface{}
		err := yaml.Unmarshal(rendered, &jsonSchema)
		if err != nil {
			message := fmt.Sprintf("error rendering yaml: %s", err)
			log.Print(message)
			internalServerError(w, message)
			return
		}
		rendered, err = json.MarshalIndent(jsonSchema, "", "  ")
		if err != nil {
			message := fmt.Sprintf("error rendering json: %s", err)
			log.Print(message)
			internalServerError(w, message)
			return
		}
	case "yaml":
		log.Printf("Rendering jsonschema in yaml")
	}

	w.Header().Set("X-EPR-JsonSchema-Format", format)
	w.Header().Set("X-EPR-JsonSchema-bytes", strconv.Itoa(jsonSchemaBytes))
	yamlHeader(w)
	fmt.Fprint(w, string(rendered))
}
