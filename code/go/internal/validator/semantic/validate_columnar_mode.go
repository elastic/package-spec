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

// ValidateColumnarModeConstraints checks data streams using logsdb_columnar or columnar
// index modes for incompatible field settings. Hard errors (doc_values: false,
// mapping-level runtime fields) always fail. Warnings (nested types, dynamic/enabled: false)
// are filterable via their SVR codes and behave as warnings when the caller uses warnOn.
func ValidateColumnarModeConstraints(fsys fspath.FS) specerrors.ValidationErrors {
	dataStreams, err := listDataStreams(fsys)
	if err != nil {
		return specerrors.ValidationErrors{specerrors.NewStructuredError(err, specerrors.UnassignedCode)}
	}

	// Identify which data streams use a columnar index mode.
	columnar := make(map[string]bool)
	for _, ds := range dataStreams {
		ok, err := isColumnarIndexMode(fsys, ds)
		if err != nil {
			return specerrors.ValidationErrors{specerrors.NewStructuredError(err, specerrors.UnassignedCode)}
		}
		columnar[ds] = ok
	}

	hasAny := false
	for _, v := range columnar {
		if v {
			hasAny = true
			break
		}
	}
	if !hasAny {
		return nil
	}

	var errs specerrors.ValidationErrors

	// Manifest-level checks (dynamic: false in index_template.mappings).
	for _, ds := range dataStreams {
		if !columnar[ds] {
			continue
		}
		errs = append(errs, checkColumnarManifest(fsys, ds)...)
	}

	// Field-level checks.
	errs = append(errs, validateFields(fsys, func(meta fieldFileMetadata, f field) specerrors.ValidationErrors {
		if !columnar[meta.dataStream] {
			return nil
		}
		return checkColumnarField(meta, f)
	})...)

	return errs
}

// isColumnarIndexMode returns true when the data stream manifest declares a columnar-family index mode.
func isColumnarIndexMode(fsys fspath.FS, dataStream string) (bool, error) {
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
	if err := yaml.Unmarshal(d, &manifest); err != nil {
		return false, fmt.Errorf("failed to parse data stream manifest in %q: %w", fsys.Path(manifestPath), err)
	}

	switch manifest.Elasticsearch.IndexMode {
	case "logsdb_columnar", "columnar":
		return true, nil
	}
	return false, nil
}

// checkColumnarManifest validates manifest-level settings for a columnar data stream.
func checkColumnarManifest(fsys fspath.FS, dataStream string) specerrors.ValidationErrors {
	manifestPath := path.Join("data_stream", dataStream, "manifest.yml")
	d, err := fs.ReadFile(fsys, manifestPath)
	if err != nil {
		return specerrors.ValidationErrors{specerrors.NewStructuredError(err, specerrors.UnassignedCode)}
	}

	var manifest struct {
		Elasticsearch struct {
			IndexTemplate struct {
				Mappings struct {
					Dynamic any `yaml:"dynamic"`
				} `yaml:"mappings"`
			} `yaml:"index_template"`
		} `yaml:"elasticsearch"`
	}
	if err := yaml.Unmarshal(d, &manifest); err != nil {
		return specerrors.ValidationErrors{specerrors.NewStructuredError(err, specerrors.UnassignedCode)}
	}

	var errs specerrors.ValidationErrors
	if dynamicIsFalse(manifest.Elasticsearch.IndexTemplate.Mappings.Dynamic) {
		errs = append(errs, specerrors.NewStructuredError(
			fmt.Errorf(`file %q is invalid: elasticsearch.index_template.mappings.dynamic is set to false; `+
				`columnar index mode has no stored _source, so unmapped fields are permanently lost when dynamic is false`,
				fsys.Path(manifestPath)),
			specerrors.CodeColumnarDynamicFalse,
		))
	}
	return errs
}

// checkColumnarField validates a single field definition for columnar compatibility.
func checkColumnarField(meta fieldFileMetadata, f field) specerrors.ValidationErrors {
	var errs specerrors.ValidationErrors

	// doc_values: false is a mapping error in columnar mode — ES rejects the index template.
	if f.DocValues != nil && !*f.DocValues {
		errs = append(errs, specerrors.NewStructuredErrorf(
			`file %q is invalid: field %q has doc_values set to false, which is rejected by Elasticsearch in columnar index mode`,
			meta.fullFilePath, f.Name,
		))
	}

	// Mapping-level runtime fields are rejected in columnar mode.
	if f.Runtime.isEnabled() {
		errs = append(errs, specerrors.NewStructuredErrorf(
			`file %q is invalid: field %q is a mapping-level runtime field, which is not supported in columnar index mode; `+
				`use a concrete field computed by an ingest pipeline, or use a search-request runtime field instead`,
			meta.fullFilePath, f.Name,
		))
	}

	// nested has limited support; deeply nested nesting is unsupported.
	if f.Type == "nested" {
		errs = append(errs, specerrors.NewStructuredError(
			fmt.Errorf(`file %q is invalid: field %q is of type "nested", which has limited support in columnar index mode; `+
				`verify that nested-in-nested usage is absent and that consumers tolerate the flattened mapping shape`,
				meta.fullFilePath, f.Name),
			specerrors.CodeColumnarNestedField,
		))
	}

	// enabled: false on an object field causes data loss — contents are not stored in columnar mode.
	if f.Enabled != nil && !*f.Enabled {
		errs = append(errs, specerrors.NewStructuredError(
			fmt.Errorf(`file %q is invalid: field %q has enabled set to false; `+
				`columnar index mode has no stored _source so disabled object contents are permanently lost`,
				meta.fullFilePath, f.Name),
			specerrors.CodeColumnarEnabledFalse,
		))
	}

	// dynamic: false on a field object causes data loss — unmapped sub-fields are not stored.
	if dynamicIsFalse(f.Dynamic) {
		errs = append(errs, specerrors.NewStructuredError(
			fmt.Errorf(`file %q is invalid: field %q has dynamic set to false; `+
				`columnar index mode has no stored _source so unmapped sub-fields are permanently lost`,
				meta.fullFilePath, f.Name),
			specerrors.CodeColumnarDynamicFalse,
		))
	}

	return errs
}

// dynamicIsFalse returns true when the YAML value represents false (bool or string).
func dynamicIsFalse(v any) bool {
	switch val := v.(type) {
	case bool:
		return !val
	case string:
		return val == "false"
	}
	return false
}
