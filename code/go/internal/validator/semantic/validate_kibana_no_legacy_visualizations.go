// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package semantic

import (
	"path"

	"github.com/elastic/kbncontent"
	"github.com/elastic/package-spec/v2/code/go/internal/fspath"
	"github.com/elastic/package-spec/v2/code/go/internal/pkgpath"
	"github.com/elastic/package-spec/v2/code/go/pkg/specerrors"
)

// ValidateKibanaNoLegacyVisualizations reports legacy Kibana visualizations in a package.
func ValidateKibanaNoLegacyVisualizations(fsys fspath.FS) specerrors.ValidationErrors {
	var errs specerrors.ValidationErrors

	// Collect by-reference visualizations for reference later.
	// Note: this does not include Lens, Maps, or Discover. That's okay for this rule because none of those are legacy
	visFilePaths := path.Join("kibana", "visualization", "*.json")
	visFiles, _ := pkgpath.Files(fsys, visFilePaths)

	for _, file := range visFiles {
		filePath := fsys.Path(file.Path())

		visJSON, err := file.Values("$")
		if err != nil {
			errs = append(errs, specerrors.NewStructuredErrorf("file \"%s\" is invalid: error getting JSON: %w", filePath, err))
			continue
		}


		visMap, ok := visJSON.(map[string]interface{})
		if !ok {
			errs = append(errs, specerrors.NewStructuredErrorf("file \"%s\" is invalid: JSON of unexpected type %T", filePath, visJSON))
			continue
		}

		desc, err := kbncontent.DescribeVisualizationSavedObject(visMap)
		if err != nil {
			errs = append(errs, specerrors.NewStructuredErrorf("file \"%s\" is invalid: error describing visualization saved object: %w", filePath, err))
			continue
		}

		if desc.IsLegacy() {
			var editor string
			if result, err := desc.Editor(); err == nil {
				editor = result
			}
			errs = append(errs, specerrors.NewStructuredErrorf("file \"%s\" is invalid: found legacy visualization \"%s\" (%s, %s)", filePath, desc.Title(), desc.SemanticType(), editor))
		}
	}

	dashboardFilePaths := path.Join("kibana", "dashboard", "*.json")
	dashboardFiles, err := pkgpath.Files(fsys, dashboardFilePaths)
	if err != nil {
		errs = append(errs, specerrors.NewStructuredErrorf("error finding Kibana dashboard files: %w", err))
		return errs
	}

	for _, file := range dashboardFiles {
		filePath := fsys.Path(file.Path())

		dashboardJSON, err := file.Values("$")
		if err != nil {
			errs = append(errs, specerrors.NewStructuredErrorf("file \"%s\" is invalid: error getting dashboard JSON: %w", filePath, err))
			continue
		}

		dashboardTitle, err := kbncontent.GetDashboardTitle(dashboardJSON)
		if err != nil {
			errs = append(errs, specerrors.NewStructuredErrorf("file \"%s\" is invalid: error fetching dashboard title: %w", filePath, err))
			continue
		}

		visualizations, err := kbncontent.DescribeByValueDashboardPanels(dashboardJSON)
		if err != nil {
			errs = append(errs, specerrors.NewStructuredErrorf("file \"%s\" is invalid: error describing dashboard panels: %w", filePath, err))
			continue
		}

		for _, desc := range visualizations {
			if desc.IsLegacy() {
				var editor string
				if result, err := desc.Editor(); err == nil {
					editor = result
				}
				err := specerrors.NewStructuredErrorf("file \"%s\" is invalid: \"%s\" contains legacy visualization: \"%s\" (%s, %s)", filePath, dashboardTitle, desc.Title(), desc.SemanticType(), editor)
				errs = append(errs, err)
			}
		}
	}

	return errs
}
