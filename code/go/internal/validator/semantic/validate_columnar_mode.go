// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package semantic

import (
	"errors"
	"fmt"
	"io/fs"
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

var (
	// Invalid combinations of elasticsearch.logsdb_columnar with other data stream settings.
	errInvalidDSTypeColumnar          = errors.New("elasticsearch.logsdb_columnar is only supported on logs data streams")
	errInvalidIndexModeLogsDBColumnar = errors.New("elasticsearch.index_mode and elasticsearch.logsdb_columnar cannot both be set; logsdb_columnar only applies when index_mode is unset")

	// Mappings that Elasticsearch rejects in columnar index modes.
	errColumnarCopyTo         = errors.New("copy_to is not supported in columnar index modes, it prevents synthetic source reconstruction; use an ingest pipeline to copy the value instead")
	errColumnarNormalizer     = errors.New("keyword normalizers other than lowercase are not supported in columnar index modes, they prevent synthetic source reconstruction; remove the normalizer or apply the transformation in an ingest pipeline")
	errColumnarRuntimeField   = errors.New("mapping-level runtime fields are not supported in columnar index modes; compute the field in an ingest pipeline or use a search-request runtime field instead")
	errColumnarNestedInNested = errors.New("columnar index modes support only a single level of nesting")

	// Mappings that drop data in columnar index modes, which have no stored _source to fall back on.
	errColumnarEnabledFalse = errors.New("enabled: false loses the object contents in columnar index modes, which have no stored _source")
	errColumnarDynamicFalse = errors.New("dynamic: false loses unmapped fields in columnar index modes, which have no stored _source")

	// Invalid index.sort settings.
	errColumnarSortOrderLength      = errors.New("index.sort.order must have one entry per index.sort.field")
	errColumnarSortFieldUndefined   = errors.New("index sort fields must be defined in the data stream fields")
	errColumnarSortFieldNoDocValues = errors.New("index sort fields must have doc values")
	errColumnarSortWithoutTimestamp = errors.New("index.sort does not include @timestamp; most queries filter on a time range, so sorting on it keeps time-range pruning effective")
)

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
// Errors without a code (invalid logsdb_columnar settings, copy_to, keyword normalizers other
// than lowercase, mapping-level runtime fields, nested inside nested, invalid index.sort fields)
// cannot be filtered. dynamic: false, enabled: false and an index.sort without @timestamp are
// reported with their own SVR codes.
func ValidateColumnarModeConstraints(fsys fspath.FS) specerrors.ValidationErrors {
	packageReadiness, err := readPackageColumnarReadiness(fsys)
	if err != nil {
		return specerrors.ValidationErrors{specerrors.NewStructuredError(err, specerrors.UnassignedCode)}
	}

	dataStreams, err := readColumnarDataStreams(fsys)
	if err != nil {
		return specerrors.ValidationErrors{specerrors.NewStructuredError(err, specerrors.UnassignedCode)}
	}

	errs := validateLogsDBColumnarSettings(dataStreams)

	ready := columnarReadyDataStreams(dataStreams, packageReadiness)
	if len(ready) == 0 {
		return errs
	}

	streamFields, fieldErrs := validateColumnarFields(fsys, ready)
	errs = append(errs, fieldErrs...)

	for _, dataStream := range ready {
		errs = append(errs, streamFields[dataStream.name].nestedInNestedErrors()...)
		errs = append(errs, validateColumnarIndexTemplate(dataStream, streamFields[dataStream.name])...)
	}

	return errs
}

// columnarDataStream is a data stream with the manifest settings the columnar checks read.
type columnarDataStream struct {
	name         string
	manifestPath string
	manifest     columnarDataStreamManifest
}

// columnarDataStreamManifest holds the data stream manifest settings that decide whether the
// columnar compatibility checks apply, and the index template settings they validate.
type columnarDataStreamManifest struct {
	Type          string `yaml:"type"`
	Elasticsearch struct {
		IndexMode      string `yaml:"index_mode"`
		LogsDBColumnar string `yaml:"logsdb_columnar"`
		IndexTemplate  struct {
			Settings struct {
				Index struct {
					Sort *columnarIndexSort `yaml:"sort"`
				} `yaml:"index"`
			} `yaml:"settings"`
			Mappings struct {
				Dynamic any `yaml:"dynamic"`
			} `yaml:"mappings"`
		} `yaml:"index_template"`
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

// readColumnarDataStreams decodes the columnar-relevant settings of every data stream manifest,
// in data stream directory order.
func readColumnarDataStreams(fsys fspath.FS) ([]columnarDataStream, error) {
	names, err := listDataStreams(fsys)
	if err != nil {
		return nil, err
	}

	dataStreams := make([]columnarDataStream, 0, len(names))
	for _, name := range names {
		manifestPath := path.Join(dataStreamDir, name, "manifest.yml")
		manifest, err := readDataStreamColumnarManifest(fsys, manifestPath)
		if err != nil {
			return nil, err
		}
		dataStreams = append(dataStreams, columnarDataStream{
			name:         name,
			manifestPath: fsys.Path(manifestPath),
			manifest:     manifest,
		})
	}
	return dataStreams, nil
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

// validateLogsDBColumnarSettings reports the data stream manifests that combine
// elasticsearch.logsdb_columnar with settings it cannot be used with.
func validateLogsDBColumnarSettings(dataStreams []columnarDataStream) specerrors.ValidationErrors {
	var errs specerrors.ValidationErrors
	for _, dataStream := range dataStreams {
		if err := validateLogsDBColumnarDataStreamType(dataStream.manifest); err != nil {
			errs = append(errs, specerrors.NewStructuredErrorf(`file "%s" is invalid: %w`, dataStream.manifestPath, err))
		}
		if err := validateLogsDBColumnarIndexMode(dataStream.manifest); err != nil {
			errs = append(errs, specerrors.NewStructuredErrorf(`file "%s" is invalid: %w`, dataStream.manifestPath, err))
		}
	}
	return errs
}

// validateLogsDBColumnarDataStreamType rejects elasticsearch.logsdb_columnar on data streams
// that are not logs data streams.
func validateLogsDBColumnarDataStreamType(manifest columnarDataStreamManifest) error {
	if manifest.Elasticsearch.LogsDBColumnar == "" || manifest.Type == logsDataStreamType {
		return nil
	}
	return fmt.Errorf("%w (data stream type is %q)", errInvalidDSTypeColumnar, manifest.Type)
}

// validateLogsDBColumnarIndexMode rejects elasticsearch.logsdb_columnar together with a fixed
// elasticsearch.index_mode.
func validateLogsDBColumnarIndexMode(manifest columnarDataStreamManifest) error {
	if manifest.Elasticsearch.LogsDBColumnar == "" || manifest.Elasticsearch.IndexMode == "" {
		return nil
	}
	return errInvalidIndexModeLogsDBColumnar
}

// columnarReadyDataStreams returns the data streams whose effective readiness is opt_in or default.
func columnarReadyDataStreams(dataStreams []columnarDataStream, packageReadiness string) []columnarDataStream {
	var ready []columnarDataStream
	for _, dataStream := range dataStreams {
		if isColumnarReady(dataStream.manifest, packageReadiness) {
			ready = append(ready, dataStream)
		}
	}
	return ready
}

// validateColumnarFields validates every field definition of the ready data streams, and
// collects them per data stream for the checks that need the whole data stream: index.sort
// and nested-in-nested.
func validateColumnarFields(fsys fspath.FS, ready []columnarDataStream) (map[string]*columnarStreamFields, specerrors.ValidationErrors) {
	streamFields := make(map[string]*columnarStreamFields, len(ready))
	for _, dataStream := range ready {
		streamFields[dataStream.name] = newColumnarStreamFields()
	}

	errs := validateFields(fsys, func(meta fieldFileMetadata, f field) specerrors.ValidationErrors {
		// Transform fields are reported with an empty data stream, so they are skipped too:
		// they are never part of a columnar data stream mapping.
		collected, ready := streamFields[meta.dataStream]
		if !ready {
			return nil
		}
		collected.add(meta, f)
		return validateColumnarField(meta, f)
	})

	return streamFields, errs
}

// validateColumnarField validates a single field definition for columnar compatibility.
func validateColumnarField(meta fieldFileMetadata, f field) specerrors.ValidationErrors {
	var errs specerrors.ValidationErrors

	// copy_to prevents synthetic source reconstruction (FieldMapper.calculateSyntheticSourceMode).
	if f.CopyTo != nil {
		errs = append(errs, columnarFieldError(meta, f, errColumnarCopyTo, specerrors.UnassignedCode))
	}

	// keyword with normalizer prevents synthetic source reconstruction (KeywordFieldMapper).
	// The built-in "lowercase" normalizer is accepted: ES defaults normalizer_skip_store_original_value
	// to true for it, so synthetic source returns the lowercased value instead of failing.
	if f.Type == "keyword" && f.Normalizer != "" && f.Normalizer != "lowercase" {
		errs = append(errs, columnarFieldError(meta, f, errColumnarNormalizer, specerrors.UnassignedCode))
	}

	if f.Runtime.isEnabled() {
		errs = append(errs, columnarFieldError(meta, f, errColumnarRuntimeField, specerrors.UnassignedCode))
	}

	if f.Enabled != nil && !*f.Enabled {
		errs = append(errs, columnarFieldError(meta, f, errColumnarEnabledFalse, specerrors.CodeColumnarEnabledFalse))
	}

	if dynamicIsFalse(f.Dynamic) {
		errs = append(errs, columnarFieldError(meta, f, errColumnarDynamicFalse, specerrors.CodeColumnarDynamicFalse))
	}

	return errs
}

func columnarFieldError(meta fieldFileMetadata, f field, err error, code string) specerrors.ValidationError {
	return specerrors.NewStructuredError(
		fmt.Errorf(`file "%s" is invalid: field %q: %w`, meta.fullFilePath, f.Name, err),
		code,
	)
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
			`file "%s" is invalid: field %q is inside nested field %q: %w`,
			child.filePath, child.name, parent, errColumnarNestedInNested,
		))
	}
	return errs
}

// sortableError returns why a field cannot back an index.sort setting, or nil when it can.
// Index sorting reads doc values, so a sort field must be defined, must keep its doc values
// and must be of a type that has them.
func (c *columnarStreamFields) sortableError(name string) error {
	info, defined := c.defined[name]
	if !defined {
		// Fleet always maps @timestamp, whether or not the package declares it. host.name is
		// only injected by Elasticsearch for its default sort, so a custom sort must map it.
		if name == "@timestamp" {
			return nil
		}
		return errColumnarSortFieldUndefined
	}
	if (info.docValues != nil && !*info.docValues) || columnarUnsortableTypes[info.fieldType] {
		return fmt.Errorf("%w (type %q)", errColumnarSortFieldNoDocValues, info.fieldType)
	}
	return nil
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

// validateColumnarIndexTemplate validates the index template settings of a columnar-ready data stream.
func validateColumnarIndexTemplate(dataStream columnarDataStream, streamFields *columnarStreamFields) specerrors.ValidationErrors {
	indexTemplate := dataStream.manifest.Elasticsearch.IndexTemplate

	var errs specerrors.ValidationErrors
	if dynamicIsFalse(indexTemplate.Mappings.Dynamic) {
		errs = append(errs, specerrors.NewStructuredError(
			fmt.Errorf(`file "%s" is invalid: elasticsearch.index_template.mappings: %w`, dataStream.manifestPath, errColumnarDynamicFalse),
			specerrors.CodeColumnarDynamicFalse,
		))
	}
	errs = append(errs, validateColumnarIndexSort(dataStream.manifestPath, indexTemplate.Settings.Index.Sort, streamFields)...)
	return errs
}

// validateColumnarIndexSort validates elasticsearch.index_template.settings.index.sort against
// the data stream field definitions. Elasticsearch does not require @timestamp in a custom sort,
// but most queries filter on a time range, so a sort that leaves it out is reported with its
// own code (SVR00016).
func validateColumnarIndexSort(manifestPath string, sort *columnarIndexSort, streamFields *columnarStreamFields) specerrors.ValidationErrors {
	if sort == nil || len(sort.Field) == 0 {
		return nil
	}

	var errs specerrors.ValidationErrors
	if len(sort.Order) > 0 && len(sort.Order) != len(sort.Field) {
		errs = append(errs, specerrors.NewStructuredErrorf(
			`file "%s" is invalid: %w (%d fields, %d orders)`,
			manifestPath, errColumnarSortOrderLength, len(sort.Field), len(sort.Order),
		))
	}

	for _, name := range sort.Field {
		if err := streamFields.sortableError(name); err != nil {
			errs = append(errs, specerrors.NewStructuredErrorf(
				`file "%s" is invalid: index.sort field %q: %w`, manifestPath, name, err,
			))
		}
	}

	if !slices.Contains(sort.Field, "@timestamp") {
		errs = append(errs, specerrors.NewStructuredError(
			fmt.Errorf(`file "%s" is invalid: %w`, manifestPath, errColumnarSortWithoutTimestamp),
			specerrors.CodeColumnarSortWithoutTimestamp,
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
