// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package semantic

import (
	"path"

	"github.com/elastic/kbncontent"
	"github.com/elastic/package-spec/v2/code/go/internal/fspath"
	"github.com/elastic/package-spec/v2/code/go/internal/pkgpath"
	se "github.com/elastic/package-spec/v2/code/go/pkg/specerrors"
)

// Reports legacy Kibana visualizations in a package.
func ValidateKibanaNoLegacyVisualizations(fsys fspath.FS) se.ValidationErrors {
	var errs se.ValidationErrors

	// Collect by-reference visualizations for reference later.
	// Note: this does not include Lens, Maps, or Discover. That's okay for this rule because none of those are legacy
	visFilePaths := path.Join("kibana", "visualization", "*.json")
	visFiles, _ := pkgpath.Files(fsys, visFilePaths)

	for _, file := range visFiles {
		visJSON, err := file.Values("$")

		if err != nil {
			errs = append(errs, se.NewStructuredErrorf("error getting JSON: %w", err))
			return errs
		}


		desc, err := kbncontent.DescribeVisualizationSavedObject(visJSON.(map[string]interface{}))
		if err != nil {
			errs = append(errs, se.NewStructuredErrorf("error describing visualization saved object: %w", err))
		}

		if desc.IsLegacy() {
			var editor string
			if result, err := desc.Editor(); err == nil {
				editor = result
			}
			errs = append(errs, se.NewStructuredErrorf("file \"%s\" is invalid: found legacy visualization \"%s\" (%s, %s)", fsys.Path(file.Path()), desc.Title(), desc.SemanticType(), editor))
		}
	}

	dashboardFilePaths := path.Join("kibana", "dashboard", "*.json")
	dashboardFiles, err := pkgpath.Files(fsys, dashboardFilePaths)
	if err != nil {
		errs = append(errs, se.NewStructuredErrorf("error finding Kibana dashboard files: %w", err))
		return errs
	}

	for _, file := range dashboardFiles {
		dashboardJSON, err := file.Values("$")
		if err != nil {
			errs = append(errs, se.NewStructuredErrorf("error getting dashboard JSON: %w", err))
			return errs
		}

		dashboardTitle, err := kbncontent.GetDashboardTitle(dashboardJSON)
		if err != nil {
			errs = append(errs, se.NewStructuredErrorf("error fetching dashboard title: %w", err))
			return errs
		}

		visualizations, err := kbncontent.DescribeByValueDashboardPanels(dashboardJSON)
		if err != nil {
			errs = append(errs, se.NewStructuredErrorf("error describing dashboard panels for %s: %w", fsys.Path(file.Path()), err))
			return errs
		}

		for _, desc := range visualizations {
			if desc.IsLegacy() {
				var editor string
				if result, err := desc.Editor(); err == nil {
					editor = result
				}
				err := se.NewStructuredErrorf("file \"%s\" is invalid: \"%s\" contains legacy visualization: \"%s\" (%s, %s)", fsys.Path(file.Path()), dashboardTitle, desc.Title(), desc.SemanticType(), editor)
				errs = append(errs, err)
			}
		}
	}

	return errs
}
