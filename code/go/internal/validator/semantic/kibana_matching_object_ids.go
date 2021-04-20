// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package semantic

import (
	"fmt"
	"path/filepath"
	"strings"

	ve "github.com/elastic/package-spec/code/go/internal/errors"

	"github.com/elastic/package-spec/code/go/internal/pkgpath"
	"github.com/pkg/errors"
)

// ValidateKibanaObjectIDs returns validation errors if there are any Kibana
// object files that define IDs not matching the file's name. That is, it returns
// validation errors if a Kibana object file, foo.json, in the package defines
// an object ID other than foo inside it.
func ValidateKibanaObjectIDs(pkgRoot string) ve.ValidationErrors {
	var errs ve.ValidationErrors

	filePaths := filepath.Join(pkgRoot, "kibana", "*", "*.json")
	objectFiles, err := pkgpath.Files(filePaths)
	if err != nil {
		errs = append(errs, errors.Wrap(err, "error finding Kibana object files"))
		return errs
	}

	for _, objectFile := range objectFiles {
		name := objectFile.Name()
		objectID, err := objectFile.Values("$.id")
		if err != nil {
			errs = append(errs, errors.Wrapf(err, "unable to get Kibana object ID in file '%s'", name))
			continue
		}

		fileID := strings.TrimRight(name, ".json")
		if fileID != objectID {
			err := fmt.Errorf("kibana object file '%s' defines non-matching ID '%s'", name, objectID)
			errs = append(errs, err)
		}
	}

	return errs
}
