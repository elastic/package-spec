// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package semantic

import (
	"fmt"
	"path"
	"strings"

	"github.com/pkg/errors"

	ve "github.com/elastic/package-spec/v2/code/go/internal/errors"
	"github.com/elastic/package-spec/v2/code/go/internal/fspath"
	"github.com/elastic/package-spec/v2/code/go/internal/pkgpath"
)

// ValidateKibanaObjectIDs returns validation errors if there are any Kibana
// object files that define IDs not matching the file's name. That is, it returns
// validation errors if a Kibana object file, foo.json, in the package defines
// an object ID other than foo inside it.
func ValidateKibanaObjectIDs(fsys fspath.FS) ve.ValidationErrors {
	var errs ve.ValidationErrors

	filePaths := path.Join("kibana", "*", "*.json")
	objectFiles, err := pkgpath.Files(fsys, filePaths)
	if err != nil {
		errs = append(errs, errors.Wrap(err, "error finding Kibana object files"))
		return errs
	}

	for _, objectFile := range objectFiles {
		filePath := objectFile.Path()

		objectID, err := objectFile.Values("$.id")
		if err != nil {
			errs = append(errs, errors.Wrapf(err, "unable to get Kibana object ID in file [%s]", fsys.Path(filePath)))
			continue
		}

		// Special case: object is of type 'security_rule'
		if path.Base(path.Dir(filePath)) == "security_rule" {
			ruleID, err := objectFile.Values("$.attributes.rule_id")
			if err != nil {
				errs = append(errs, errors.Wrapf(err, "unable to get rule ID in file [%s]", fsys.Path(filePath)))
				continue
			}

			objectIDValue, ok := objectID.(string)
			if !ok {
				errs = append(errs, errors.Wrap(err, "expect object ID to be a string"))
				continue
			}

			ruleIDValue, ok := ruleID.(string)
			if !ok {
				errs = append(errs, errors.Wrap(err, "expect rule ID to be a string"))
				continue
			}

			if !strings.HasPrefix(objectIDValue, ruleIDValue) {
				err := fmt.Errorf("kibana object ID [%s] should start with rule ID [%s]", objectIDValue, ruleIDValue)
				errs = append(errs, err)
				continue
			}
		}

		// fileID == filename without the extension == expected ID of Kibana object defined inside file.
		fileName := path.Base(filePath)
		fileExt := path.Ext(filePath)
		fileID := strings.Replace(fileName, fileExt, "", -1)
		if fileID != objectID {
			err := fmt.Errorf("kibana object file [%s] defines non-matching ID [%s]", fsys.Path(filePath), objectID)
			errs = append(errs, err)
		}
	}

	return errs
}
