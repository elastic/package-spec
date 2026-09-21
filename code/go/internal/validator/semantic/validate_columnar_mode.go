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

// columnarUnsupportedTypes is the set of field types that have no synthetic-source
// implementation in Elasticsearch. Indexing such a field into a columnar data stream
// causes ES to reject the index template at PUT time.
// Source: FieldMapper.calculateSyntheticSourceMode / IndexMode.validateAllFieldsReconstructableFromDocValues.
//
// This check is defensive: none of these types are currently allowed by the `type` enum in
// spec/integration/data_stream/fields/fields.spec.yml, so JSON-schema validation rejects them
// before this semantic validator runs. It exists to guard against a future enum expansion
// accidentally admitting a type that columnar mode cannot store.
var columnarUnsupportedTypes = map[string]bool{
	"completion":         true,
	"search_as_you_type": true,
	"token_count":        true,
	"rank_feature":       true,
	"rank_features":      true,
	"percolator":         true,
}

// ValidateColumnarModeConstraints checks data streams using logsdb_columnar or columnar
// index modes, or declaring elasticsearch.columnar.supported, for incompatible field
// settings. Hard errors (doc_values: false, mapping-level runtime fields, copy_to,
// keyword+normalizer, unsupported types) always fail. Warnings (nested types,
// dynamic/enabled: false) are filterable via their SVR codes and behave as warnings
// when the caller uses warnOn.
func ValidateColumnarModeConstraints(fsys fspath.FS) specerrors.ValidationErrors {
	dataStreams, err := listDataStreams(fsys)
	if err != nil {
		return specerrors.ValidationErrors{specerrors.NewStructuredError(err, specerrors.UnassignedCode)}
	}

	// Identify which data streams require columnar compatibility checks.
	columnar := make(map[string]bool)
	for _, ds := range dataStreams {
		ok, err := isColumnarEnabled(fsys, ds)
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

// isColumnarEnabled returns true when the data stream manifest declares a columnar-family
// index mode, or opts in to columnar readiness with elasticsearch.columnar.supported: true.
// The readiness flag lets a package declare that its mappings are columnar-compatible (so
// Fleet can offer the per-data-stream opt-in) while keeping its default index mode; the
// compatibility checks must run in that case too.
func isColumnarEnabled(fsys fspath.FS, dataStream string) (bool, error) {
	manifestPath := path.Join("data_stream", dataStream, "manifest.yml")
	d, err := fs.ReadFile(fsys, manifestPath)
	if err != nil {
		return false, fmt.Errorf("failed to read data stream manifest in %q: %w", fsys.Path(manifestPath), err)
	}

	var manifest struct {
		Elasticsearch struct {
			IndexMode string `yaml:"index_mode"`
			Columnar  struct {
				Supported bool `yaml:"supported"`
			} `yaml:"columnar"`
		} `yaml:"elasticsearch"`
	}
	if err := yaml.Unmarshal(d, &manifest); err != nil {
		return false, fmt.Errorf("failed to parse data stream manifest in %q: %w", fsys.Path(manifestPath), err)
	}

	if manifest.Elasticsearch.Columnar.Supported {
		return true, nil
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
	// A `columnar.doc_values` override replaces the base setting when the index mode is
	// columnar, so evaluate the effective value rather than the declared one. This is what
	// lets a package keep an ECS-imported field such as event.original (doc_values: false)
	// and re-enable doc values for columnar mode only.
	docValues := f.DocValues
	var columnarDocValues *bool
	if f.Columnar != nil {
		columnarDocValues = f.Columnar.DocValues
		if columnarDocValues != nil {
			docValues = columnarDocValues
		}
	}
	switch {
	case columnarDocValues != nil && !*columnarDocValues:
		// Belt and braces: the JSON schema only allows `columnar.doc_values: true`, but an
		// explicit false here would be a no-op override that still breaks the mapping.
		errs = append(errs, specerrors.NewStructuredErrorf(
			`file %q is invalid: field %q sets columnar.doc_values to false; columnar index mode requires doc values, only true is allowed`,
			meta.fullFilePath, f.Name,
		))
	case docValues != nil && !*docValues:
		errs = append(errs, specerrors.NewStructuredErrorf(
			`file %q is invalid: field %q has doc_values set to false, which is rejected by Elasticsearch in columnar index mode`,
			meta.fullFilePath, f.Name,
		))
	}

	// f.Columnar.Index is intentionally not validated: an inverted index is allowed in
	// columnar mode (it only costs storage), so both true and false pass without a warning.

	// copy_to prevents synthetic source reconstruction (FieldMapper.calculateSyntheticSourceMode).
	if f.CopyTo != nil {
		errs = append(errs, specerrors.NewStructuredErrorf(
			`file %q is invalid: field %q has copy_to set, which prevents synthetic source reconstruction in columnar index mode; `+
				`use an ingest pipeline to copy the value instead`,
			meta.fullFilePath, f.Name,
		))
	}

	// keyword with normalizer prevents synthetic source reconstruction (KeywordFieldMapper).
	// The built-in "lowercase" normalizer is accepted: ES defaults normalizer_skip_store_original_value
	// to true for it, so synthetic source returns the lowercased value instead of failing.
	if f.Type == "keyword" && f.Normalizer != "" && f.Normalizer != "lowercase" {
		errs = append(errs, specerrors.NewStructuredErrorf(
			`file %q is invalid: field %q is a keyword field with a normalizer, which prevents synthetic source reconstruction in columnar index mode; `+
				`remove the normalizer or apply the transformation in an ingest pipeline`,
			meta.fullFilePath, f.Name,
		))
	}

	// Types with no synthetic-source implementation are rejected by ES at template PUT time.
	if columnarUnsupportedTypes[f.Type] {
		errs = append(errs, specerrors.NewStructuredErrorf(
			`file %q is invalid: field %q is of type %q, which is not supported in columnar index mode`,
			meta.fullFilePath, f.Name, f.Type,
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
