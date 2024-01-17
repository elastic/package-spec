// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package semantic

import (
	"encoding/json"
	"errors"
	"fmt"
	"path"

	"github.com/elastic/kbncontent"
	"github.com/mitchellh/mapstructure"

	"github.com/elastic/package-spec/v3/code/go/internal/fspath"
	"github.com/elastic/package-spec/v3/code/go/internal/pkgpath"
	"github.com/elastic/package-spec/v3/code/go/pkg/specerrors"
)

var (
	errDashboardPanelWithoutFilter   = errors.New("at least one panel does not have a filter")
	errDashboardWithQueryAndNoFilter = errors.New("saved query found, but no filter")
	errDashboardFilterNotFound       = errors.New("no filter found")
)

// ValidateKibanaFilterPresent checks that all the dashboards included in a package
// contain a filter, so only data related to its datasets is queried.
func ValidateKibanaFilterPresent(fsys fspath.FS) specerrors.ValidationErrors {
	var errs specerrors.ValidationErrors

	filePaths := path.Join("kibana", "dashboard", "*.json")
	dashboardFiles, err := pkgpath.Files(fsys, filePaths)
	if err != nil {
		errs = append(errs, specerrors.NewStructuredErrorf("error finding Kibana dashboard files: %w", err))
		return errs
	}
	for _, file := range dashboardFiles {
		err = checkDashboardHasFilter(file)
		if err != nil {
			code := specerrors.CodeKibanaDashboardWithoutFilter
			if errors.Is(err, errDashboardWithQueryAndNoFilter) {
				code = specerrors.CodeKibanaDashboardWithQueryButNoFilter
			}
			errs = append(errs,
				specerrors.NewStructuredError(
					fmt.Errorf("file \"%s\" is invalid: expected filter in dashboard: %w", fsys.Path(file.Path()), err),
					code),
			)
		}
	}

	return errs
}

func checkDashboardHasFilter(file pkgpath.File) error {
	err := findPanelsFilters(file)
	if err != nil {
		dashboardErr := findDashboardFilter(file)
		if dashboardErr != nil {
			return fmt.Errorf("%w and %w", dashboardErr, err)
		}
	}
	return nil
}

func findPanelsFilters(file pkgpath.File) error {
	dashboardJSON, err := file.Values("$")
	if err != nil {
		return fmt.Errorf("unable to get dashboard document: %w", err)
	}

	visualizations, err := kbncontent.DescribeByValueDashboardPanels(dashboardJSON)
	if err != nil {
		return fmt.Errorf("error describing visualization saved object: %w", err)
	}

	for _, visualization := range visualizations {
		if visualization.CanUseFilter() && !visualization.HasFilters() {
			return errDashboardPanelWithoutFilter
		}
	}

	return nil
}

func findDashboardFilter(file pkgpath.File) error {
	searchJSON, err := file.Values("$.attributes.kibanaSavedObjectMeta.searchSourceJSON")
	if err != nil {
		return fmt.Errorf("unable to find search definition: %w", err)
	}

	var search struct {
		Filter []any `mapstructure:"filter"`
		Query  struct {
			Query string `mapstructure:"query"`
		} `mapstructure:"query"`
	}
	switch searchJSON := searchJSON.(type) {
	case map[string]any:
		// Dashboard is decoded, as in source packages.
		err = mapstructure.Decode(searchJSON, &search)
		if err != nil {
			return fmt.Errorf("unable to decode search definition: %w", err)
		}
	case string:
		// Dashboard is encoded, as in built packages.
		err = json.Unmarshal([]byte(searchJSON), &search)
		if err != nil {
			return fmt.Errorf("unable to decode embedded search definition: %w", err)
		}
	default:
		return fmt.Errorf("unexpected type for search source JSON: %T", searchJSON)
	}

	if len(search.Filter) == 0 {
		if len(search.Query.Query) > 0 {
			return errDashboardWithQueryAndNoFilter
		}
		return errDashboardFilterNotFound
	}

	return nil
}
