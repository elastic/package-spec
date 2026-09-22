// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package semantic

import (
	"fmt"
	"io/fs"
	"path"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/elastic/package-spec/v3/code/go/internal/fspath"
	"github.com/elastic/package-spec/v3/code/go/pkg/specerrors"
)

type useOtelSuffixStream struct {
	Input string `yaml:"input"`
}

type useOtelSuffixManifest struct {
	UseOtelSuffix bool                  `yaml:"use_otel_suffix"`
	Streams       []useOtelSuffixStream `yaml:"streams"`
}

// ValidateUseOtelSuffix rejects use_otel_suffix on data streams that define an input.
// The flag is only for data streams that ship mappings and pipelines without inputs.
// Data streams that need an input should use the otelcol input type, which already
// applies the .otel index pattern suffix.
func ValidateUseOtelSuffix(fsys fspath.FS) specerrors.ValidationErrors {
	dataStreamNames, err := listDataStreams(fsys)
	if err != nil {
		return specerrors.ValidationErrors{specerrors.NewStructuredError(err, specerrors.UnassignedCode)}
	}

	var errs specerrors.ValidationErrors
	for _, dataStreamName := range dataStreamNames {
		manifestPath := path.Join(dataStreamDir, dataStreamName, "manifest.yml")
		data, err := fs.ReadFile(fsys, manifestPath)
		if err != nil {
			errs = append(errs, specerrors.NewStructuredErrorf(
				"file %q is invalid: failed to read manifest: %w", fsys.Path(manifestPath), err))
			continue
		}

		var manifest useOtelSuffixManifest
		if err := yaml.Unmarshal(data, &manifest); err != nil {
			errs = append(errs, specerrors.NewStructuredErrorf(
				"file %q is invalid: failed to parse manifest: %w", fsys.Path(manifestPath), err))
			continue
		}

		if !manifest.UseOtelSuffix {
			continue
		}

		var inputs []string
		for _, stream := range manifest.Streams {
			if stream.Input == "" {
				continue
			}
			inputs = append(inputs, fmt.Sprintf("%q", stream.Input))
		}
		if len(inputs) == 0 {
			continue
		}

		errs = append(errs, specerrors.NewStructuredError(fmt.Errorf(
			"file %q is invalid: use_otel_suffix is only allowed on data streams without inputs (inputs defined: %s). If this data stream needs an input, use the otelcol input type instead, which already applies the .otel index pattern suffix",
			fsys.Path(manifestPath), strings.Join(inputs, ", ")), specerrors.CodeUseOtelSuffixWithInput))
	}

	return errs
}
