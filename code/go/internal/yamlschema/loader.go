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
	"sync"

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

type fileSchemaKey struct {
	schemaPath  string
	specVersion string
}

// FileSchemaLoader loads and caches compiled JSON schemas. The cache avoids
// re-compiling schemas across multiple package validations: santhosh-tekuri
// re-parses every $ref'd YAML file from scratch on each compilation, which is
// expensive. The same (schemaPath, specVersion) pair always produces the same
// compiled schema, so caching is safe.
type FileSchemaLoader struct {
	cache sync.Map
}

func NewFileSchemaLoader() *FileSchemaLoader {
	return &FileSchemaLoader{}
}

func (l *FileSchemaLoader) Load(fsys fs.FS, schemaPath string, options spectypes.FileSchemaLoadOptions) (spectypes.FileSchema, error) {
	key := fileSchemaKey{schemaPath, options.SpecVersion.Original()}
	if cached, ok := l.cache.Load(key); ok {
		return cached.(*FileSchema), nil
	}

	var state validationState
	formats := newFormatCheckers(&state)

	c := jsonschema.NewCompiler()
	c.DefaultDraft(jsonschema.Draft7)
	c.AssertFormat()
	c.UseLoader(&fsysLoader{fsys: fsys, version: options.SpecVersion})
	for _, f := range formats {
		c.RegisterFormat(f)
	}

	schema, err := c.Compile("file:///" + schemaPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load schema for %q: %v", schemaPath, err)
	}
	result := &FileSchema{schema: schema, state: &state, options: options}
	l.cache.Store(key, result)
	return result, nil
}

type FileSchema struct {
	schema  *jsonschema.Schema
	state   *validationState
	mu      sync.Mutex // serializes access to state during concurrent Validate calls
	options spectypes.FileSchemaLoadOptions
}

func (s *FileSchema) Validate(fsys fs.FS, filePath string) specerrors.ValidationErrors {
	data, err := loadItemSchema(fsys, filePath, s.options.ContentType, s.options.SpecVersion)
	if err != nil {
		return specerrors.ValidationErrors{specerrors.NewStructuredError(err, specerrors.UnassignedCode)}
	}

	instance, err := decodeJSON(data)
	if err != nil {
		return specerrors.ValidationErrors{specerrors.NewStructuredError(
			fmt.Errorf("decoding item file failed: %w", err), specerrors.UnassignedCode)}
	}

	s.mu.Lock()
	s.state.fsys = fsys
	s.state.currentPath = path.Dir(filePath)
	s.state.sizeLimit = s.options.Limits.MaxRelativePathSize()
	defer func() {
		s.state.fsys = nil
		s.mu.Unlock()
	}()

	verr := s.schema.Validate(instance)
	if verr == nil {
		return nil
	}

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
