// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package semantic

import (
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path"
	"slices"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/elastic/package-spec/v3/code/go/internal/fspath"
	"github.com/elastic/package-spec/v3/code/go/pkg/specerrors"
)

// Values of elasticsearch.logsdb_columnar that make a logs data stream columnar-ready. The
// remaining data stream value, "unsupported", keeps the data stream on LogsDB, like an absent
// value, so it is not listed here.
const (
	// columnarOptIn offers the user an opt-in; Elasticsearch defaults stay in effect until then.
	columnarOptIn = "opt_in"
	// columnarDefault makes new installations use logsdb_columnar; the user can opt out.
	columnarDefault = "default"
)

// logsDataStreamType is the only data stream type that supports the logsdb_columnar index mode.
const logsDataStreamType = "logs"

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

// columnarUnsortableTypes is the set of field types that cannot back an index.sort setting,
// because they have no doc values Lucene can sort segments on.
var columnarUnsortableTypes = map[string]bool{
	"text":            true,
	"match_only_text": true,
	"wildcard":        true,
	"flattened":       true,
	"nested":          true,
	"object":          true,
	"group":           true,
}

// ValidateColumnarModeConstraints checks the logs data streams that declare readiness for the
// logsdb_columnar index mode through elasticsearch.logsdb_columnar, at package or data stream
// level, for mappings and index sort settings Elasticsearch rejects in columnar index modes.
//
// Hard errors (copy_to, keyword normalizers other than lowercase, mapping-level runtime fields,
// nested inside nested, types without synthetic source, invalid index.sort) always fail.
// Warnings (dynamic: false, enabled: false) are filterable via their SVR codes and behave as
// warnings when the caller uses warnOn.
func ValidateColumnarModeConstraints(fsys fspath.FS) specerrors.ValidationErrors {
	dataStreams, err := listDataStreams(fsys)
	if err != nil {
		return specerrors.ValidationErrors{specerrors.NewStructuredError(err, specerrors.UnassignedCode)}
	}

	packageReadiness, err := readPackageColumnarReadiness(fsys)
	if err != nil {
		return specerrors.ValidationErrors{specerrors.NewStructuredError(err, specerrors.UnassignedCode)}
	}

	var errs specerrors.ValidationErrors

	// Resolve which data streams are ready for the columnar index mode, and report the
	// data stream manifests that combine logsdb_columnar with incompatible settings.
	ready := make(map[string]bool, len(dataStreams))
	manifestPaths := make(map[string]string, len(dataStreams))
	for _, dataStream := range dataStreams {
		manifestPath := path.Join(dataStreamDir, dataStream, "manifest.yml")
		manifestPaths[dataStream] = manifestPath

		manifest, err := readDataStreamColumnarManifest(fsys, manifestPath)
		if err != nil {
			return specerrors.ValidationErrors{specerrors.NewStructuredError(err, specerrors.UnassignedCode)}
		}

		if manifest.Elasticsearch.LogsDBColumnar != "" {
			if manifest.Type != logsDataStreamType {
				errs = append(errs, specerrors.NewStructuredErrorf(
					`file "%s" is invalid: elasticsearch.logsdb_columnar is only supported on logs data streams (data stream type is %q)`,
					fsys.Path(manifestPath), manifest.Type,
				))
			}
			if manifest.Elasticsearch.IndexMode != "" {
				errs = append(errs, specerrors.NewStructuredErrorf(
					`file "%s" is invalid: elasticsearch.index_mode and elasticsearch.logsdb_columnar cannot both be set; `+
						`logsdb_columnar only applies when index_mode is unset`,
					fsys.Path(manifestPath),
				))
			}
		}

		ready[dataStream] = isColumnarReady(manifest, packageReadiness)
	}

	if !slices.Contains(slices.Collect(maps.Values(ready)), true) {
		return errs
	}

	// Manifest-level checks (dynamic: false, index_template mapping overrides, index.sort).
	// Field definitions are collected first: index.sort and nested-in-nested need the whole
	// data stream, not a single field.
	streamFields := make(map[string]*columnarStreamFields, len(dataStreams))
	errs = append(errs, validateFields(fsys, func(meta fieldFileMetadata, f field) specerrors.ValidationErrors {
		// Transform fields are reported with an empty data stream; they are never part of a
		// columnar data stream mapping.
		if meta.transform != "" || meta.dataStream == "" || !ready[meta.dataStream] {
			return nil
		}
		collected, ok := streamFields[meta.dataStream]
		if !ok {
			collected = newColumnarStreamFields()
			streamFields[meta.dataStream] = collected
		}
		collected.add(meta, f)
		return checkColumnarField(meta, f)
	})...)

	for _, dataStream := range slices.Sorted(maps.Keys(ready)) {
		if !ready[dataStream] {
			continue
		}
		collected := streamFields[dataStream]
		if collected == nil {
			collected = newColumnarStreamFields()
		}
		errs = append(errs, collected.nestedInNestedErrors()...)
		errs = append(errs, checkColumnarManifest(fsys, manifestPaths[dataStream], collected)...)
	}

	return errs
}

// columnarDataStreamManifest holds the data stream manifest settings that decide whether the
// columnar compatibility checks apply.
type columnarDataStreamManifest struct {
	Type          string `yaml:"type"`
	Elasticsearch struct {
		IndexMode      string `yaml:"index_mode"`
		LogsDBColumnar string `yaml:"logsdb_columnar"`
	} `yaml:"elasticsearch"`
}

// isColumnarReady reports whether a data stream must be validated for columnar compatibility.
// The effective readiness is the data stream value when set, and the package value otherwise.
// Only logs data streams without a fixed index_mode can be ready.
func isColumnarReady(manifest columnarDataStreamManifest, packageReadiness string) bool {
	if manifest.Type != logsDataStreamType || manifest.Elasticsearch.IndexMode != "" {
		return false
	}
	readiness := manifest.Elasticsearch.LogsDBColumnar
	if readiness == "" {
		readiness = packageReadiness
	}
	return readiness == columnarOptIn || readiness == columnarDefault
}

// readPackageColumnarReadiness returns the package-level elasticsearch.logsdb_columnar value,
// or an empty string when the root manifest does not declare it.
func readPackageColumnarReadiness(fsys fspath.FS) (string, error) {
	d, err := fs.ReadFile(fsys, "manifest.yml")
	if errors.Is(err, os.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("failed to read manifest in %q: %w", fsys.Path("manifest.yml"), err)
	}

	var manifest struct {
		Elasticsearch struct {
			LogsDBColumnar string `yaml:"logsdb_columnar"`
		} `yaml:"elasticsearch"`
	}
	if err := yaml.Unmarshal(d, &manifest); err != nil {
		return "", fmt.Errorf("failed to parse manifest in %q: %w", fsys.Path("manifest.yml"), err)
	}
	return manifest.Elasticsearch.LogsDBColumnar, nil
}

// readDataStreamColumnarManifest decodes the columnar-relevant settings of a data stream manifest.
func readDataStreamColumnarManifest(fsys fspath.FS, manifestPath string) (columnarDataStreamManifest, error) {
	var manifest columnarDataStreamManifest

	d, err := fs.ReadFile(fsys, manifestPath)
	if errors.Is(err, os.ErrNotExist) {
		return manifest, nil
	}
	if err != nil {
		return manifest, fmt.Errorf("failed to read data stream manifest in %q: %w", fsys.Path(manifestPath), err)
	}
	if err := yaml.Unmarshal(d, &manifest); err != nil {
		return manifest, fmt.Errorf("failed to parse data stream manifest in %q: %w", fsys.Path(manifestPath), err)
	}
	return manifest, nil
}

// columnarFieldInfo is the subset of a field definition needed to validate index.sort.
type columnarFieldInfo struct {
	fieldType string
	docValues *bool
}

// columnarNestedField records where a `nested` field was declared, to report nested-in-nested.
type columnarNestedField struct {
	name     string
	filePath string
}

// columnarStreamFields accumulates the field definitions of a single data stream across all of
// its fields files, so that checks spanning the whole data stream can run once per data stream.
type columnarStreamFields struct {
	// defined maps the full dotted field name to the information index.sort needs.
	defined map[string]columnarFieldInfo
	// nested lists the `nested` fields in declaration order, for deterministic reporting.
	nested []columnarNestedField
}

func newColumnarStreamFields() *columnarStreamFields {
	return &columnarStreamFields{defined: map[string]columnarFieldInfo{}}
}

func (c *columnarStreamFields) add(meta fieldFileMetadata, f field) {
	c.defined[f.Name] = columnarFieldInfo{fieldType: f.Type, docValues: f.DocValues}
	if f.Type == "nested" {
		c.nested = append(c.nested, columnarNestedField{name: f.Name, filePath: meta.fullFilePath})
	}
}

// nestedInNestedErrors reports every `nested` field declared under another `nested` field.
// Columnar index modes support only a single level of nesting. Both shapes are detected,
// because the field walker reports both with their full dotted name: a `nested` field nested
// in another one through `fields`, and separate entries such as `a` and `a.b`, possibly
// declared in different fields files of the same data stream.
func (c *columnarStreamFields) nestedInNestedErrors() specerrors.ValidationErrors {
	paths := make(map[string]bool, len(c.nested))
	for _, n := range c.nested {
		paths[n.name] = true
	}

	var errs specerrors.ValidationErrors
	for _, child := range c.nested {
		parent := ""
		for candidate := range paths {
			if !strings.HasPrefix(child.name, candidate+".") {
				continue
			}
			// Report the closest nested ancestor.
			if len(candidate) > len(parent) {
				parent = candidate
			}
		}
		if parent == "" {
			continue
		}
		errs = append(errs, specerrors.NewStructuredErrorf(
			`file "%s" is invalid: field %q is a nested field inside another nested field (%q); `+
				`columnar index modes support only a single level of nesting`,
			child.filePath, child.name, parent,
		))
	}
	return errs
}

// columnarMappingNode is a single node of elasticsearch.index_template.mappings.properties.
// Only the mapping parameters that block columnar mode are decoded.
type columnarMappingNode struct {
	CopyTo     any                            `yaml:"copy_to"`
	Properties map[string]columnarMappingNode `yaml:"properties"`
	Fields     map[string]columnarMappingNode `yaml:"fields"`
}

// columnarIndexSort is the elasticsearch.index_template.settings.index.sort section. Both
// `field` and `order` accept a single string or a list of strings.
type columnarIndexSort struct {
	Field columnarStringList `yaml:"field"`
	Order columnarStringList `yaml:"order"`
}

// columnarStringList decodes a YAML value that is either a string or a list of strings.
type columnarStringList []string

func (l *columnarStringList) UnmarshalYAML(value *yaml.Node) error {
	var list []string
	if err := value.Decode(&list); err == nil {
		*l = list
		return nil
	}
	var single string
	if err := value.Decode(&single); err != nil {
		return err
	}
	*l = []string{single}
	return nil
}

// checkColumnarManifest validates the manifest-level settings of a columnar-ready data stream.
func checkColumnarManifest(fsys fspath.FS, manifestPath string, streamFields *columnarStreamFields) specerrors.ValidationErrors {
	d, err := fs.ReadFile(fsys, manifestPath)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return specerrors.ValidationErrors{specerrors.NewStructuredError(err, specerrors.UnassignedCode)}
	}

	var manifest struct {
		Elasticsearch struct {
			IndexTemplate struct {
				Settings struct {
					Index struct {
						Sort *columnarIndexSort `yaml:"sort"`
					} `yaml:"index"`
				} `yaml:"settings"`
				Mappings struct {
					Dynamic    any                            `yaml:"dynamic"`
					Properties map[string]columnarMappingNode `yaml:"properties"`
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
			fmt.Errorf(`file "%s" is invalid: elasticsearch.index_template.mappings.dynamic is set to false; `+
				`columnar index mode has no stored _source, so unmapped fields are permanently lost when dynamic is false`,
				fsys.Path(manifestPath)),
			specerrors.CodeColumnarDynamicFalse,
		))
	}

	// Fleet merges elasticsearch.index_template.mappings on top of the mappings generated
	// from fields/, so blockers declared here are not caught by the field-level checks.
	// Note that since spec 3.0.0 the JSON schema itself rejects arbitrary `properties` in
	// this section, so this walk is mostly defence in depth (build-mode manifests and any
	// future relaxation of the schema).
	errs = append(errs, checkColumnarMappingProperties(
		fsys.Path(manifestPath),
		"elasticsearch.index_template.mappings.properties",
		manifest.Elasticsearch.IndexTemplate.Mappings.Properties,
	)...)

	errs = append(errs, checkColumnarIndexSort(
		fsys.Path(manifestPath),
		manifest.Elasticsearch.IndexTemplate.Settings.Index.Sort,
		streamFields,
	)...)

	return errs
}

// checkColumnarMappingProperties walks a properties map from an index_template mapping
// override and reports the settings that Elasticsearch rejects in columnar index mode.
func checkColumnarMappingProperties(manifestPath, parentPath string, properties map[string]columnarMappingNode) specerrors.ValidationErrors {
	var errs specerrors.ValidationErrors
	for _, name := range slices.Sorted(maps.Keys(properties)) {
		node := properties[name]
		nodePath := parentPath + "." + name

		if node.CopyTo != nil {
			errs = append(errs, specerrors.NewStructuredErrorf(
				`file "%s" is invalid: %s has copy_to set, which prevents synthetic source reconstruction in columnar index mode; `+
					`use an ingest pipeline to copy the value instead`,
				manifestPath, nodePath,
			))
		}

		errs = append(errs, checkColumnarMappingProperties(manifestPath, nodePath+".properties", node.Properties)...)
		errs = append(errs, checkColumnarMappingProperties(manifestPath, nodePath+".fields", node.Fields)...)
	}
	return errs
}

// checkColumnarIndexSort validates elasticsearch.index_template.settings.index.sort against the
// data stream field definitions. Index sorting reads doc values, so every sort field must be
// defined, must keep its doc values and must be of a type that has them. Columnar index modes
// always sort on @timestamp, so it has to be part of the sort specification.
func checkColumnarIndexSort(manifestPath string, sort *columnarIndexSort, streamFields *columnarStreamFields) specerrors.ValidationErrors {
	if sort == nil || len(sort.Field) == 0 {
		return nil
	}

	var errs specerrors.ValidationErrors
	if len(sort.Order) > 0 && len(sort.Order) != len(sort.Field) {
		errs = append(errs, specerrors.NewStructuredErrorf(
			`file "%s" is invalid: index.sort.order has %d entries but index.sort.field has %d, they must have the same length`,
			manifestPath, len(sort.Order), len(sort.Field),
		))
	}

	for _, name := range sort.Field {
		info, defined := streamFields.defined[name]
		if !defined {
			// Fleet always maps @timestamp as a date field, whether or not the package declares it.
			if name == "@timestamp" {
				continue
			}
			errs = append(errs, specerrors.NewStructuredErrorf(
				`file "%s" is invalid: index.sort field %q is not defined in the data stream fields`,
				manifestPath, name,
			))
			continue
		}
		noDocValues := info.docValues != nil && !*info.docValues
		if noDocValues || columnarUnsortableTypes[info.fieldType] {
			errs = append(errs, specerrors.NewStructuredErrorf(
				`file "%s" is invalid: index.sort field %q has no doc values (type %q), index sorting requires doc values`,
				manifestPath, name, info.fieldType,
			))
		}
	}

	if !slices.Contains(sort.Field, "@timestamp") {
		errs = append(errs, specerrors.NewStructuredErrorf(
			`file "%s" is invalid: index.sort must include @timestamp`,
			manifestPath,
		))
	}

	return errs
}

// checkColumnarField validates a single field definition for columnar compatibility.
func checkColumnarField(meta fieldFileMetadata, f field) specerrors.ValidationErrors {
	var errs specerrors.ValidationErrors

	// copy_to prevents synthetic source reconstruction (FieldMapper.calculateSyntheticSourceMode).
	if f.CopyTo != nil {
		errs = append(errs, specerrors.NewStructuredErrorf(
			`file "%s" is invalid: field %q has copy_to set, which prevents synthetic source reconstruction in columnar index mode; `+
				`use an ingest pipeline to copy the value instead`,
			meta.fullFilePath, f.Name,
		))
	}

	// keyword with normalizer prevents synthetic source reconstruction (KeywordFieldMapper).
	// The built-in "lowercase" normalizer is accepted: ES defaults normalizer_skip_store_original_value
	// to true for it, so synthetic source returns the lowercased value instead of failing.
	if f.Type == "keyword" && f.Normalizer != "" && f.Normalizer != "lowercase" {
		errs = append(errs, specerrors.NewStructuredErrorf(
			`file "%s" is invalid: field %q is a keyword field with a normalizer, which prevents synthetic source reconstruction in columnar index mode; `+
				`remove the normalizer or apply the transformation in an ingest pipeline`,
			meta.fullFilePath, f.Name,
		))
	}

	// Types with no synthetic-source implementation are rejected by ES at template PUT time.
	if columnarUnsupportedTypes[f.Type] {
		errs = append(errs, specerrors.NewStructuredErrorf(
			`file "%s" is invalid: field %q is of type %q, which is not supported in columnar index mode`,
			meta.fullFilePath, f.Name, f.Type,
		))
	}

	// Mapping-level runtime fields are rejected in columnar mode.
	if f.Runtime.isEnabled() {
		errs = append(errs, specerrors.NewStructuredErrorf(
			`file "%s" is invalid: field %q is a mapping-level runtime field, which is not supported in columnar index mode; `+
				`use a concrete field computed by an ingest pipeline, or use a search-request runtime field instead`,
			meta.fullFilePath, f.Name,
		))
	}

	// enabled: false on an object field causes data loss — contents are not stored in columnar mode.
	if f.Enabled != nil && !*f.Enabled {
		errs = append(errs, specerrors.NewStructuredError(
			fmt.Errorf(`file "%s" is invalid: field %q has enabled set to false; `+
				`columnar index mode has no stored _source so disabled object contents are permanently lost`,
				meta.fullFilePath, f.Name),
			specerrors.CodeColumnarEnabledFalse,
		))
	}

	// dynamic: false on a field object causes data loss — unmapped sub-fields are not stored.
	if dynamicIsFalse(f.Dynamic) {
		errs = append(errs, specerrors.NewStructuredError(
			fmt.Errorf(`file "%s" is invalid: field %q has dynamic set to false; `+
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
