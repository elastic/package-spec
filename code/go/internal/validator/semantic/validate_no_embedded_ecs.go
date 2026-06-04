// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package semantic

import (
	"io/fs"
	"path"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/elastic/package-spec/v3/code/go/internal/fspath"
	"github.com/elastic/package-spec/v3/code/go/pkg/specerrors"
)

// embeddedEcsDataStreamManifest is a minimal struct for unmarshalling just the
// elasticsearch.index_template.mappings.dynamic_templates section of a data stream manifest.
type embeddedEcsDataStreamManifest struct {
	Elasticsearch struct {
		IndexTemplate struct {
			Mappings struct {
				DynamicTemplates []map[string]any `yaml:"dynamic_templates"`
			} `yaml:"mappings"`
		} `yaml:"index_template"`
	} `yaml:"elasticsearch"`
}

// ValidateNoEmbeddedEcsInDynamicTemplates rejects data stream manifests that contain
// dynamic_templates entries whose keys match the "^_embedded_ecs" pattern.
//
// Keys starting with "_embedded_ecs" are auto-injected by elastic-package at build time
// when import_mappings is enabled. They must not appear in source packages and are
// rejected in source validation mode. Built packages (where these keys are expected)
// should be validated with ModeBuild, not ModeSource.
func ValidateNoEmbeddedEcsInDynamicTemplates(fsys fspath.FS) specerrors.ValidationErrors {
	dataStreams, err := listDataStreams(fsys)
	if err != nil {
		return specerrors.ValidationErrors{specerrors.NewStructuredError(err, specerrors.UnassignedCode)}
	}

	var errs specerrors.ValidationErrors
	for _, dataStream := range dataStreams {
		manifestPath := path.Join(dataStreamDir, dataStream, "manifest.yml")
		streamErrs := checkDataStreamManifestForEmbeddedEcs(fsys, manifestPath)
		errs = append(errs, streamErrs...)
	}
	return errs
}

func checkDataStreamManifestForEmbeddedEcs(fsys fspath.FS, manifestPath string) specerrors.ValidationErrors {
	data, err := fs.ReadFile(fsys, manifestPath)
	if err != nil {
		return specerrors.ValidationErrors{
			specerrors.NewStructuredErrorf("file \"%s\" is invalid: failed to read manifest: %w", fsys.Path(manifestPath), err),
		}
	}

	var manifest embeddedEcsDataStreamManifest
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		return specerrors.ValidationErrors{
			specerrors.NewStructuredErrorf("file \"%s\" is invalid: failed to parse manifest: %w", fsys.Path(manifestPath), err),
		}
	}

	var errs specerrors.ValidationErrors
	for _, entry := range manifest.Elasticsearch.IndexTemplate.Mappings.DynamicTemplates {
		for key := range entry {
			if strings.HasPrefix(key, "_embedded_ecs") {
				errs = append(errs, specerrors.NewStructuredErrorf(
					"file \"%s\" is invalid: dynamic template %q starts with \"_embedded_ecs\"; this key is auto-injected at build time and must not appear in source packages",
					fsys.Path(manifestPath), key,
				))
			}
		}
	}
	return errs
}
