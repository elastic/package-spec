// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package semantic

import (
	"encoding/json"
	"fmt"
	"log"
	"path"
	"strings"

	"github.com/pkg/errors"

	ve "github.com/elastic/package-spec/code/go/internal/errors"
	"github.com/elastic/package-spec/code/go/internal/fspath"
	"github.com/elastic/package-spec/code/go/internal/pkgpath"
)

type reference struct {
	ID   string
	Name string
	Type string
}

// ValidateVisualizationsUsedByValue warns if there are any Kibana
// Dashboard that defines visualizations by reference instead of value.
// That is, it warns if a Kibana dashbaord file, foo.json,
// defines some visualization using reference (containing an element of
// "visualization" type inside references key).
func ValidateVisualizationsUsedByValue(fsys fspath.FS) ve.ValidationErrors {
	var errs ve.ValidationErrors

	filePaths := path.Join("kibana", "dashboard", "*.json")
	objectFiles, err := pkgpath.Files(fsys, filePaths)
	if err != nil {
		errs = append(errs, errors.Wrap(err, "error finding Kibana Dashboard files"))
		return errs
	}

	for _, objectFile := range objectFiles {
		filePath := objectFile.Path()

		objectReferences, err := objectFile.Values(`$.references`)
		if err != nil {
			// no references key in dashboard json
			// errs = append(errs, errors.Wrapf(err, "unable to get Kibana Dashboard references in file [%s]", fsys.Path(filePath)))
			continue
		}

		ids, err := anyReference(objectReferences, fsys.Path(filePath))
		if err != nil {
			errs = append(errs, errors.Wrap(err, "error getting references"))
		}
		if len(ids) > 0 {
			log.Printf("Warning: visualization by reference found in %s: %s", filePath, strings.Join(ids, ", "))
		}
	}

	return errs
}

func anyReference(val interface{}, path string) ([]string, error) {
	allReferences, err := toReferenceSlice(val)
	if err != nil {
		return []string{}, fmt.Errorf("unable to convert references in file [%s]: %w", path, err)
	}

	if len(allReferences) == 0 {
		return []string{}, nil
	}

	var idReferences []string
	for _, reference := range allReferences {
		switch reference.Type {
		case "lens", "visualization":
			log.Printf("Warning: %s by reference found: %s (dashboard %s)", reference.Type, reference.ID, path)
			idReferences = append(idReferences, reference.ID)
		}
	}
	return idReferences, nil

}

func toReferenceSlice(val interface{}) ([]reference, error) {
	var refs []reference
	jsonbody, err := json.Marshal(val)
	if err != nil {
		log.Printf("error encoding reference list: %s", err)
		return refs, nil
	}

	err = json.Unmarshal(jsonbody, &refs)
	if err != nil {
		log.Printf("error unmarshaling references: %s", err)
		return refs, nil
	}

	return refs, nil
}
