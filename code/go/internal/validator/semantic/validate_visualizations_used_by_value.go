// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package semantic

import (
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

		objectReferences, err := objectFile.Values(`$.references`)
		if err != nil {
			// no references key in dashboard json
			continue
		}

		references, err := anyReference(objectReferences)
		if err != nil {
			errs = append(errs, errors.Wrap(err, "error getting references in file: %s", fsys.Path(filePath)))
		}
		if len(references) > 0 {
			s := fmt.Sprintf("%s (%s)", references[0].ID, references[0].Type)
			for _, ref := range references[1:] {
				s = fmt.Sprintf("%s, %s (%s)", s, ref.ID, ref.Type)
			}
			log.Printf("Warning: references found in dashboard %s: %s", filePath, s)
		}
	}

	return errs
}

func anyReference(val interface{}) ([]reference, error) {
	allReferences, err := toReferenceSlice(val)
	if err != nil {
		return []reference{}, fmt.Errorf("unable to convert references: %w", err)
	}

	if len(allReferences) == 0 {
		return []reference{}, nil
	}

	var references []reference
	for _, reference := range allReferences {
		switch reference.Type {
		case "lens", "visualization":
			references = append(references, reference)
		}
	}
	return references, nil

}

func toReferenceSlice(val interface{}) ([]reference, error) {
	vals, ok := val.([]interface{})
	if !ok {
		return nil, errors.New("conversion error to array")
	}
	var refs []reference
	for _, v := range vals {
		r, ok := v.(map[string]interface{})
		if !ok {
			return nil, errors.New("conversion error to reference element")
		}
		ref := reference{
			ID:   r["id"].(string),
			Type: r["type"].(string),
			Name: r["name"].(string),
		}

		refs = append(refs, ref)
	}
	return refs, nil
}
