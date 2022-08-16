// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package semantic

import (
	"encoding/json"
	"fmt"
	"log"
	"path"

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

		objectReferences, err := objectFile.Values(`$.references[?(@.type=="visualization")]`)
		if err != nil {
			errs = append(errs, errors.Wrapf(err, "unable to get Kibana Dashboard references in file [%s]", fsys.Path(filePath)))
			continue
		}

		anyReference(objectReferences, fsys.Path(filePath))
	}

	return errs
}

func anyReference(val interface{}, path string) (bool, error) {
	references, err := toReferenceSlice(val)
	if err != nil {
		return false, fmt.Errorf("unable to convert references in file [%s]", path)
	}

	if len(references) == 0 {
		return false, nil
	}

	log.Printf("Warning: Kibana Dashboard %s contains visualizations by reference:", path)
	for _, reference := range references {
		log.Printf("Warning: visualization by reference found: %s", reference.ID)
	}
	return true, nil

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
