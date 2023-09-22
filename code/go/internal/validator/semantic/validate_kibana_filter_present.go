// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package semantic

import (
	"errors"
	"fmt"
	"path"

	"github.com/mitchellh/mapstructure"

	"github.com/elastic/package-spec/v2/code/go/internal/fspath"
	"github.com/elastic/package-spec/v2/code/go/internal/pkgpath"
	ve "github.com/elastic/package-spec/v2/code/go/pkg/errors"
)

// ValidateKibanaFilterPresent checks that all the dashboards included in a package
// contain a filter, so only data related to its datasets is queried.
func ValidateKibanaFilterPresent(fsys fspath.FS) ve.ValidationErrors {
	var errs ve.ValidationErrors

	filePaths := path.Join("kibana", "dashboard", "*.json")
	dashboardFiles, err := pkgpath.Files(fsys, filePaths)
	if err != nil {
		errs = append(errs, ve.NewStructuredError(fmt.Errorf("error finding Kibana dashboard files: %w", err), ve.TODO_code))
		return errs
	}
	for _, file := range dashboardFiles {
		err = checkDashboardHasFilter(file)
		if err != nil {
			errs = append(errs,
				ve.NewStructuredError(
					fmt.Errorf("file \"%s\" is invalid: expected filter in dashboard: %w", fsys.Path(file.Path()), err),
					ve.TODO_code),
			)
		}
	}

	return errs
}

func checkDashboardHasFilter(file pkgpath.File) error {
	searchJSON, err := file.Values("$.attributes.kibanaSavedObjectMeta.searchSourceJSON")
	if err != nil {
		return fmt.Errorf("unable to find search definition: %w", err)
	}

	var search struct {
		Filter []interface{} `mapstructure:"filter"`
		Query  struct {
			Query string `mapstructure:"query"`
		} `mapstructure:"query"`
	}
	err = mapstructure.Decode(searchJSON, &search)
	if err != nil {
		return fmt.Errorf("unable to decode search definition: %w", err)
	}

	if len(search.Filter) == 0 {
		if len(search.Query.Query) > 0 {
			return errors.New("saved query found, but no filter")
		}
		return errors.New("no filter found")
	}

	return nil
}
