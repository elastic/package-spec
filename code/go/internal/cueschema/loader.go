// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cueschema

import (
	"bytes"
	"fmt"
	"io/fs"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/load"
	cueyaml "cuelang.org/go/pkg/encoding/yaml"

	spec "github.com/elastic/package-spec"
	ve "github.com/elastic/package-spec/code/go/internal/errors"
	"github.com/elastic/package-spec/code/go/internal/spectypes"
	"github.com/elastic/package-spec/code/go/internal/yamlschema"
)

// FileSchemaLoader implements schema loading from CUE definitions.
type FileSchemaLoader struct{}

// NewFileSchemaLoader creates a new FileSchemaLoader for CUE definitions.
func NewFileSchemaLoader() *FileSchemaLoader {
	return &FileSchemaLoader{}
}

// Load loads a schema from a CUE file in the given filesystem.
func (f *FileSchemaLoader) Load(fsys fs.FS, schemaPath string, options spectypes.FileSchemaLoadOptions) (spectypes.FileSchema, error) {
	parts := strings.SplitN(schemaPath, "#", 2)

	filePath := parts[0]
	definition := ""
	if len(parts) > 1 {
		definition = "#" + parts[1]
	}

	d, err := fs.ReadFile(fsys, filePath)
	if err != nil {
		return nil, err
	}

	spec, err := loadSpec(d, options)
	if err != nil {
		return nil, fmt.Errorf("failed to load instance with spec: %w", err)
	}

	if definition != "" {
		spec = spec.LookupDef(definition)
		if err := spec.Err(); err != nil {
			return nil, fmt.Errorf("failed to find CUE definition %q in %s: %w", definition, filePath, err)
		}
	}

	return &FileSchema{spec, options}, nil
}

// FileSchema is a schema for a file.
type FileSchema struct {
	spec    cue.Value
	options spectypes.FileSchemaLoadOptions
}

// Validate validates that the given file complies with the schema.
func (s *FileSchema) Validate(fsys fs.FS, filePath string) ve.ValidationErrors {
	d, err := fs.ReadFile(fsys, filePath)
	if err != nil {
		return ve.ValidationErrors{err}
	}

	d, err = yamlschema.ConvertYAMLToJSON(d)
	if err != nil {
		return ve.ValidationErrors{err}
	}

	expr, err := cueyaml.Unmarshal(d)
	if err != nil {
		return ve.ValidationErrors{
			fmt.Errorf("failed to parse yaml file %q: %w", filePath, err),
		}
	}

	v := s.spec.Context().BuildExpr(expr, cue.Filename(filePath))
	v = v.Unify(s.spec)
	errs := v.Validate(cue.Concrete(true))
	if errs != nil {
		return ve.ValidationErrors(validationErrors(filePath, errs))
	}

	return nil
}

func loadSpec(specBytes []byte, options spectypes.FileSchemaLoadOptions) (cue.Value, error) {
	// This is a hack till https://github.com/cue-lang/cue/issues/607 is solved.
	tmpDir, err := os.MkdirTemp("", "package-spec-")
	if err != nil {
		return cue.Value{}, fmt.Errorf("failed to create tmp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	files := []string{
		"cue.mod/module.cue",
		"definitions.cue",
	}

	specFS := spec.FS()
	for _, f := range files {
		d, err := fs.ReadFile(specFS, f)
		if err != nil {
			return cue.Value{}, fmt.Errorf("failed to read %q", f)
		}
		dstPath := filepath.Join(tmpDir, f)
		os.MkdirAll(filepath.Dir(dstPath), 0755)
		err = ioutil.WriteFile(dstPath, d, 0644)
		if err != nil {
			return cue.Value{}, fmt.Errorf("failed to write %q for copy of definitions", dstPath)
		}
	}

	var specBuffer bytes.Buffer
	specBuffer.Write(specBytes)
	if sv := options.SpecVersion; sv != nil {
		// This way of injection is a bit hacky, but tags require that all defined tags are used.
		specBuffer.WriteString("\n")
		fmt.Fprintf(&specBuffer, "spec_version_major: %d\n", sv.Major())
		fmt.Fprintf(&specBuffer, "spec_version_minor: %d\n", sv.Major())
		fmt.Fprintf(&specBuffer, "spec_version_patch: %d\n", sv.Major())
		fmt.Fprintf(&specBuffer, "spec_version_prerelease: \"%s\"\n", sv.Prerelease())
		fmt.Fprintf(&specBuffer, "spec_version: \"%s\"\n", sv.String())
	}

	specFilePath := filepath.Join(tmpDir, "spec.cue")
	err = ioutil.WriteFile(specFilePath, specBuffer.Bytes(), 0644)
	if err != nil {
		return cue.Value{}, fmt.Errorf("failed to write %q for copy of spec", specFilePath)
	}

	instances := load.Instances([]string{specFilePath}, &load.Config{
		Dir: tmpDir,
	})
	if len(instances) != 1 {
		return cue.Value{}, fmt.Errorf("only 1 instance expected, found %d", len(instances))
	}

	cueCtx := cuecontext.New()
	v := cueCtx.BuildInstance(instances[0])
	return v, v.Err()
}
