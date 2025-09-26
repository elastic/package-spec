// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package semantic

import (
	"io/fs"
	"path"

	"gopkg.in/yaml.v3"

	"github.com/elastic/package-spec/v3/code/go/internal/fspath"
	"github.com/elastic/package-spec/v3/code/go/pkg/specerrors"
)

// ValidateStreamTemplates validates that all referenced template_path files exist for data streams
func ValidateStreamTemplates(fsys fspath.FS) specerrors.ValidationErrors {
	var errs specerrors.ValidationErrors

	dataStreams, err := listDataStreams(fsys)
	if err != nil {
		return specerrors.ValidationErrors{specerrors.NewStructuredError(err, specerrors.UnassignedCode)}
	}

	for _, dataStream := range dataStreams {
		manifestPath := path.Join("data_stream", dataStream, "manifest.yml")
		streamErrs := validateDataStreamManifestTemplates(fsys, manifestPath, dataStream)
		errs = append(errs, streamErrs...)
	}

	return errs
}

func validateDataStreamManifestTemplates(fsys fspath.FS, manifestPath, dataStreamName string) specerrors.ValidationErrors {
	var errs specerrors.ValidationErrors

	data, err := fs.ReadFile(fsys, manifestPath)
	if err != nil {
		return specerrors.ValidationErrors{specerrors.NewStructuredErrorf("file \"%s\" is invalid: failed to read manifest: %w", fsys.Path(manifestPath), err)}
	}

	var manifest struct {
		Streams []struct {
			Input        string `yaml:"input"`
			TemplatePath string `yaml:"template_path"`
		} `yaml:"streams"`
	}

	err = yaml.Unmarshal(data, &manifest)
	if err != nil {
		return specerrors.ValidationErrors{specerrors.NewStructuredErrorf("file \"%s\" is invalid: failed to parse manifest: %w", fsys.Path(manifestPath), err)}
	}

	for _, stream := range manifest.Streams {
		if stream.TemplatePath == "" {
			continue // template_path is optional
		}

		// Check if template file exists
		templatePath := path.Join("data_stream", dataStreamName, "agent", "stream", stream.TemplatePath)
		_, err := fs.Stat(fsys, templatePath)
		if err != nil {
			errs = append(errs, specerrors.NewStructuredErrorf(
				"file \"%s\" is invalid: stream \"%s\" references template_path \"%s\" but file \"%s\" does not exist",
				fsys.Path(manifestPath), stream.Input, stream.TemplatePath, fsys.Path(templatePath)))
		}
	}

	return errs
}
