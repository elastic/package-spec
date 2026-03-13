// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package yamlschema

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"path"
	"strings"

	"github.com/Masterminds/semver/v3"
	jsonschema "github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/santhosh-tekuri/jsonschema/v6/kind"
	"golang.org/x/text/language"
	"golang.org/x/text/message"

	"gopkg.in/yaml.v3"

	"github.com/elastic/package-spec/v3/code/go/internal/spectypes"
	"github.com/elastic/package-spec/v3/code/go/pkg/specerrors"
)

var defaultPrinter = message.NewPrinter(language.English)

var semver3_0_0 = semver.MustParse("3.0.0")

// FileSchemaLoader creates a single shared jsonschema.Compiler for the given
// spec FS and version. The compiler caches parsed schema content internally,
// so all Load calls within the same loader avoid re-parsing $ref'd files.
// A new FileSchemaLoader is created per LoadSpec call, and Validate calls
// within a single validation are always sequential, so no locking is needed.
type FileSchemaLoader struct {
	compiler    *jsonschema.Compiler
	specVersion semver.Version
	fsys        fs.FS
	vocabulary  *elasticVocabulary
}

func NewFileSchemaLoader(fsys fs.FS, specVersion semver.Version) *FileSchemaLoader {
	l := &FileSchemaLoader{specVersion: specVersion, fsys: fsys}
	l.vocabulary = newElasticVocabulary()
	c := jsonschema.NewCompiler()
	c.DefaultDraft(jsonschema.Draft7)
	c.UseLoader(&fsysLoader{fsys: fsys, version: specVersion})
	c.RegisterVocabulary(l.vocabulary.vocab)
	l.compiler = c
	return l
}

func (l *FileSchemaLoader) Load(schemaPath string, options spectypes.FileSchemaLoadOptions) (spectypes.FileSchema, error) {
	fs := &FileSchema{loader: l, options: options}

	schema, err := l.compiler.Compile("file:///" + schemaPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load schema for %q: %v", schemaPath, err)
	}
	fs.schema = schema
	return fs, nil
}

// FS returns the spec filesystem this loader was initialized with.
func (l *FileSchemaLoader) FS() fs.FS { return l.fsys }

// Version returns the spec version this loader was initialized with.
func (l *FileSchemaLoader) Version() semver.Version { return l.specVersion }

type FileSchema struct {
	schema  *jsonschema.Schema
	loader  *FileSchemaLoader
	options spectypes.FileSchemaLoadOptions
}

func (s *FileSchema) Validate(fsys fs.FS, filePath string) specerrors.ValidationErrors {
	data, err := loadItemSchema(fsys, filePath, s.options.ContentType, s.loader.specVersion)
	if err != nil {
		return specerrors.ValidationErrors{specerrors.NewStructuredError(err, specerrors.UnassignedCode)}
	}

	instance, err := decodeJSON(data)
	if err != nil {
		return specerrors.ValidationErrors{specerrors.NewStructuredError(
			fmt.Errorf("decoding item file failed: %w", err), specerrors.UnassignedCode)}
	}

	s.loader.vocabulary.setContext(fsys, path.Dir(filePath), s.options.Limits.MaxRelativePathSize())
	defer s.loader.vocabulary.clearContext()

	verr := s.schema.Validate(instance)
	if verr != nil {
		ve, ok := verr.(*jsonschema.ValidationError)
		if !ok {
			return specerrors.ValidationErrors{specerrors.NewStructuredError(verr, specerrors.UnassignedCode)}
		}

		var errs specerrors.ValidationErrors
		collectLeafErrors(ve, &errs)
		if len(errs) == 0 {
			// Fallback for top-level validation failures with no sub-errors.
			errs = append(errs, specerrors.NewStructuredError(verr, specerrors.UnassignedCode))
		}
		return errs
	}

	return nil
}

// collectLeafErrors traverses the ValidationError tree and appends only leaf
// errors (those with no causes) to errs, skipping transparent Reference
// wrappers. For Contains failures it picks only the most-relevant element
// error (the one with the deepest instance-location path) to avoid reporting
// every non-matching array element.
func collectLeafErrors(ve *jsonschema.ValidationError, errs *specerrors.ValidationErrors) {
	if len(ve.Causes) == 0 {
		// Skip oneOf "too many matched" errors — semantic validators handle those.
		if k, ok := ve.ErrorKind.(*kind.OneOf); ok && len(k.Subschemas) > 0 {
			return
		}

		loc := strings.Join(ve.InstanceLocation, ".")
		if loc == "" {
			loc = "(root)"
		}

		// Split multi-value Required errors into individual per-property errors.
		if k, ok := ve.ErrorKind.(*kind.Required); ok && len(k.Missing) > 1 {
			for _, m := range k.Missing {
				*errs = append(*errs, specerrors.NewStructuredErrorf("field %s: missing property '%s'", loc, m))
			}
			return
		}

		// Split multi-value AdditionalProperties errors into individual per-property errors.
		if k, ok := ve.ErrorKind.(*kind.AdditionalProperties); ok && len(k.Properties) > 1 {
			for _, p := range k.Properties {
				*errs = append(*errs, specerrors.NewStructuredErrorf("field %s: additional properties '%s' not allowed", loc, p))
			}
			return
		}

		msg := ve.ErrorKind.LocalizedString(defaultPrinter)
		*errs = append(*errs, specerrors.NewStructuredErrorf("field %s: %s", loc, msg))
		return
	}

	switch ve.ErrorKind.(type) {
	case *kind.Reference:
		// Transparent $ref wrapper — skip directly to its single cause.
		for _, c := range ve.Causes {
			collectLeafErrors(c, errs)
		}
	case *kind.Contains, *kind.OneOf, *kind.AnyOf:
		// For contains/oneOf/anyOf failures each cause is one branch that did
		// not satisfy the sub-schema. Report only the branch closest to
		// matching (deepest instance location = fewest missing fields).
		best := deepestLeaf(ve.Causes)
		if best != nil {
			collectLeafErrors(best, errs)
		}
	default:
		for _, c := range ve.Causes {
			collectLeafErrors(c, errs)
		}
	}
}

// deepestLeaf returns the cause whose deepest leaf instance-location is the
// longest. On a tie the first candidate (lowest array index) wins.
func deepestLeaf(causes []*jsonschema.ValidationError) *jsonschema.ValidationError {
	var best *jsonschema.ValidationError
	bestDepth := -1
	for _, c := range causes {
		d := maxLeafDepth(c)
		if d > bestDepth {
			bestDepth = d
			best = c
		}
	}
	return best
}

// maxLeafDepth returns the length of the longest instance-location found in
// any leaf of the subtree rooted at ve.
func maxLeafDepth(ve *jsonschema.ValidationError) int {
	if len(ve.Causes) == 0 {
		return len(ve.InstanceLocation)
	}
	max := len(ve.InstanceLocation)
	for _, c := range ve.Causes {
		if d := maxLeafDepth(c); d > max {
			max = d
		}
	}
	return max
}

func loadItemSchema(fsys fs.FS, filePath string, contentType *spectypes.ContentType, specVersion semver.Version) ([]byte, error) {
	data, err := fs.ReadFile(fsys, filePath)
	if err != nil {
		return nil, specerrors.ValidationErrors{specerrors.NewStructuredErrorf("reading item file failed: %w", err)}
	}
	if contentType != nil && contentType.MediaType == "application/x-yaml" {
		return convertYAMLToJSON(data, specVersion.LessThan(semver3_0_0))
	}
	return data, nil
}

func convertYAMLToJSON(data []byte, expandKeys bool) ([]byte, error) {
	var c interface{}
	err := yaml.Unmarshal(data, &c)
	if err != nil {
		return nil, fmt.Errorf("unmarshalling YAML file failed: %w", err)
	}
	if expandKeys {
		c = expandItemKey(c)
	}

	data, err = json.Marshal(&c)
	if err != nil {
		return nil, fmt.Errorf("converting YAML to JSON failed: %w", err)
	}
	return data, nil
}

func expandItemKey(c interface{}) interface{} {
	if c == nil {
		return c
	}

	// c is an array
	if cArr, isArray := c.([]interface{}); isArray {
		arr := []interface{}{}
		for _, ca := range cArr {
			arr = append(arr, expandItemKey(ca))
		}
		return arr
	}

	// c is map[string]interface{}
	if cMap, isMapString := c.(map[string]interface{}); isMapString {
		expanded := MapStr{}
		for k, v := range cMap {
			ex := expandItemKey(v)
			_, err := expanded.Put(k, ex)
			if err != nil {
				panic(fmt.Errorf("unexpected error while setting key value (key: %s): %w", k, err))
			}
		}
		return expanded
	}
	return c // c is something else, e.g. string, int, etc.
}

func decodeJSON(data []byte) (any, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	var v any
	if err := dec.Decode(&v); err != nil {
		return nil, err
	}
	return v, nil
}
