// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package semantic

import (
	"fmt"
	"io/fs"
	"path"

	"gopkg.in/yaml.v3"

	"github.com/elastic/package-spec/v3/code/go/internal/fspath"
	"github.com/elastic/package-spec/v3/code/go/pkg/specerrors"
)

// ValidateDimensionsPresent verifies if dimension fields are of one of the expected types.
func ValidateDimensionsPresent(fsys fspath.FS) specerrors.ValidationErrors {
	dimensionPresent := make(map[string]struct{})
	errs := validateFields(fsys, func(metadata fieldFileMetadata, f field) specerrors.ValidationErrors {
		if f.Dimension {
			dimensionPresent[metadata.dataStream] = struct{}{}
		}
		return nil
	})
	if len(errs) > 0 {
		return errs
	}

	dataStreams, err := listDataStreams(fsys)
	if err != nil {
		return specerrors.ValidationErrors{specerrors.NewStructuredError(err, specerrors.UnassignedCode)}
	}
	for _, dataStream := range dataStreams {
		tsEnabled, err := isTimeSeriesModeEnabled(fsys, dataStream)
		if err != nil {
			return specerrors.ValidationErrors{specerrors.NewStructuredError(err, specerrors.UnassignedCode)}
		}
		_, hasDimensions := dimensionPresent[dataStream]
		if tsEnabled && !hasDimensions {
			errs = append(errs, specerrors.NewStructuredErrorf(
				`file "%s" is invalid: time series mode enabled but no dimensions configured`,
				fsys.Path("data_stream", dataStream, "manifest.yml"),
			))
		}
	}
	return errs
}

func isTimeSeriesModeEnabled(fsys fspath.FS, dataStream string) (bool, error) {
	manifestPath := path.Join("data_stream", dataStream, "manifest.yml")
	d, err := fs.ReadFile(fsys, manifestPath)
	if err != nil {
		return false, fmt.Errorf("failed to read data stream manifest in %q: %w", fsys.Path(manifestPath), err)
	}

	var manifest struct {
		Elasticsearch struct {
			IndexMode string `yaml:"index_mode"`
		} `yaml:"elasticsearch"`
	}
	err = yaml.Unmarshal(d, &manifest)
	if err != nil {
		return false, fmt.Errorf("failed to parse data stream manifest in %q: %w", fsys.Path(manifestPath), err)
	}

	return manifest.Elasticsearch.IndexMode == "time_series", nil
}
