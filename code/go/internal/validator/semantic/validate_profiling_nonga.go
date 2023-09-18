// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package semantic

import (
	"fmt"
	"io/fs"
	"path"

	"github.com/Masterminds/semver/v3"
	"gopkg.in/yaml.v3"

	ve "github.com/elastic/package-spec/v2/code/go/internal/errors"
	"github.com/elastic/package-spec/v2/code/go/internal/fspath"
	pve "github.com/elastic/package-spec/v2/code/go/pkg/errors"
)

// ValidateProfilingNonGA validates that the profiling data type is not used in GA packages,
// as this data type is in technical preview and can be eventually removed.
func ValidateProfilingNonGA(fsys fspath.FS) pve.ValidationErrors {
	manifestVersion, err := readManifestVersion(fsys)
	if err != nil {
		vError := ve.NewStructuredError(err, "manifest.yml", "", ve.Critical)
		return pve.ValidationErrors{vError}
	}

	semVer, err := semver.NewVersion(manifestVersion)
	if err != nil {
		vError := ve.NewStructuredError(err, "manifest.yml", "", ve.Critical)
		return pve.ValidationErrors{vError}
	}

	if semVer.Major() == 0 || semVer.Prerelease() != "" {
		return nil
	}

	dataStreams, err := listDataStreams(fsys)
	if err != nil {
		vError := ve.NewStructuredError(err, "data_stream", "", ve.Critical)
		return pve.ValidationErrors{vError}
	}

	var errs pve.ValidationErrors
	for _, dataStream := range dataStreams {
		err := validateProfilingTypeNotUsed(fsys, dataStream)
		if err != nil {
			errs = append(errs, err)
		}
	}
	return errs
}

func validateProfilingTypeNotUsed(fsys fspath.FS, dataStream string) pve.ValidationError {
	manifestPath := path.Join("data_stream", dataStream, "manifest.yml")
	d, err := fs.ReadFile(fsys, manifestPath)
	if err != nil {
		return ve.NewStructuredError(
			fmt.Errorf("failed to read data stream manifest in \"%s\": %w", fsys.Path(manifestPath), err),
			manifestPath,
			"",
			ve.Critical)
	}

	var manifest struct {
		Type string `yaml:"type"`
	}
	err = yaml.Unmarshal(d, &manifest)
	if err != nil {
		return ve.NewStructuredError(
			fmt.Errorf("failed to parse data stream manifest in \"%s\": %w", fsys.Path(manifestPath), err),
			manifestPath,
			"",
			ve.Critical)
	}

	if manifest.Type == "profiling" {
		return ve.NewStructuredError(
			fmt.Errorf("file \"%s\" is invalid: profiling data type cannot be used in GA packages", fsys.Path(manifestPath)),
			manifestPath,
			"",
			ve.Critical)
	}

	return nil
}
