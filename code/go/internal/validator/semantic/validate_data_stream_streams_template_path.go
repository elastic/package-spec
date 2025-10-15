// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package semantic

import (
	"errors"
	"io/fs"
	"path"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/elastic/package-spec/v3/code/go/internal/fspath"
	"github.com/elastic/package-spec/v3/code/go/pkg/specerrors"
)

const (
	defaultStreamTemplatePath = "stream.yml.hbs"
)

var (
	errFailedToReadManifest  = errors.New("failed to read manifest")
	errFailedToParseManifest = errors.New("failed to parse manifest")
	errTemplateNotFound      = errors.New("template file not found")
)

// ValidateStreamTemplates validates that all referenced template_path files exist for data streams
func ValidateStreamTemplates(fsys fspath.FS) specerrors.ValidationErrors {
	var errs specerrors.ValidationErrors

	dataStreams, err := listDataStreams(fsys)
	if err != nil {
		return specerrors.ValidationErrors{specerrors.NewStructuredError(err, specerrors.UnassignedCode)}
	}

	for _, dataStream := range dataStreams {
		streamErrs := validateDataStreamManifestTemplates(fsys, dataStream)
		errs = append(errs, streamErrs...)
	}

	return errs
}

func validateDataStreamManifestTemplates(fsys fspath.FS, dataStreamName string) specerrors.ValidationErrors {
	var errs specerrors.ValidationErrors

	manifestPath := path.Join("data_stream", dataStreamName, "manifest.yml")
	data, err := fs.ReadFile(fsys, manifestPath)
	if err != nil {
		return specerrors.ValidationErrors{specerrors.NewStructuredErrorf("file \"%s\" is invalid: %w", fsys.Path(manifestPath), errFailedToReadManifest)}
	}

	var manifest struct {
		Streams []struct {
			Input        string `yaml:"input"`
			TemplatePath string `yaml:"template_path"`
		} `yaml:"streams"`
	}

	err = yaml.Unmarshal(data, &manifest)
	if err != nil {
		return specerrors.ValidationErrors{specerrors.NewStructuredErrorf("file \"%s\" is invalid: %w", fsys.Path(manifestPath), errFailedToParseManifest)}
	}

	for _, stream := range manifest.Streams {
		streamPath := stream.TemplatePath
		if stream.TemplatePath == "" {
			// When no template_path is specified, it defaults to "stream.yml.hbs"
			streamPath = defaultStreamTemplatePath
		}

		// Walk through the "data_stream/<dataStreamName>/agent/stream" directory
		// This mirrors the logic in fleet where the assets are filtered based on the template_path
		// https://github.com/elastic/kibana/blob/main/x-pack/platform/plugins/shared/fleet/server/services/package_policy.ts#L3317
		streamDir := path.Join("data_stream", dataStreamName, "agent", "stream")
		found := false
		fs.WalkDir(fsys, streamDir, func(filePath string, d fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if !d.IsDir() && path.Base(filePath) != "" && strings.HasSuffix(filePath, streamPath) {
				found = true
				return fs.SkipDir // Stop walking once found
			}
			return nil
		})
		if !found {
			if stream.TemplatePath == "" {
				errs = append(errs, specerrors.NewStructuredErrorf(
					"file \"%s\" is invalid: stream \"%s\" is missing template file \"%s\": %w",
					fsys.Path(manifestPath), stream.Input, streamPath, errTemplateNotFound))
				continue
			}
			errs = append(errs, specerrors.NewStructuredErrorf(
				"file \"%s\" is invalid: stream \"%s\" references template_path \"%s\": %w",
				fsys.Path(manifestPath), stream.Input, streamPath, errTemplateNotFound))
			continue
		}
	}

	return errs
}
