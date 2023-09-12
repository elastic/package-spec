// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package yamlschema

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"path"
	"sync"

	"github.com/Masterminds/semver/v3"
	"github.com/elastic/gojsonschema"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"

	ve "github.com/elastic/package-spec/v2/code/go/internal/errors"
	"github.com/elastic/package-spec/v2/code/go/internal/spectypes"
)

var semver3_0_0 = semver.MustParse("3.0.0")

type FileSchemaLoader struct{}

func NewFileSchemaLoader() *FileSchemaLoader {
	return &FileSchemaLoader{}
}

func (*FileSchemaLoader) Load(fs fs.FS, schemaPath string, options spectypes.FileSchemaLoadOptions) (spectypes.FileSchema, error) {
	schemaLoader := NewReferenceLoaderFileSystem("file:///"+schemaPath, fs, options.SpecVersion)
	schema, err := gojsonschema.NewSchema(schemaLoader)
	if err != nil {
		return nil, fmt.Errorf("failed to load schema for %q: %v", schemaPath, err)
	}
	return &FileSchema{schema, options}, nil
}

type FileSchema struct {
	schema  *gojsonschema.Schema
	options spectypes.FileSchemaLoadOptions
}

var formatCheckersMutex sync.Mutex

func (s *FileSchema) Validate(fsys fs.FS, filePath string) ve.ValidationErrors {
	data, err := loadItemSchema(fsys, filePath, s.options.ContentType, s.options.SpecVersion)
	if err != nil {
		return ve.ValidationErrors{err}
	}

	formatCheckersMutex.Lock()
	defer func() {
		unloadRelativePathFormatChecker()
		unloadDataStreamNameFormatChecker()
		formatCheckersMutex.Unlock()
	}()

	loadRelativePathFormatChecker(fsys, path.Dir(filePath), s.options.Limits.MaxRelativePathSize())
	loadDataStreamNameFormatChecker(fsys, path.Dir(filePath))
	result, err := s.schema.Validate(gojsonschema.NewBytesLoader(data))
	if err != nil {
		return ve.ValidationErrors{err}
	}

	if !result.Valid() {
		var errs ve.ValidationErrors
		for _, re := range result.Errors() {
			errs = append(errs, fmt.Errorf("field %s: %s", re.Field(), adjustErrorDescription(re.Description())))
		}
		return errs
	}

	return nil // item content is valid according to the loaded schema
}

func loadItemSchema(fsys fs.FS, path string, contentType *spectypes.ContentType, specVersion semver.Version) ([]byte, error) {
	data, err := fs.ReadFile(fsys, path)
	if err != nil {
		return nil, ve.ValidationErrors{fmt.Errorf("reading item file failed: %w", err)}
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
		return nil, errors.Wrapf(err, "unmarshalling YAML file failed")
	}
	if expandKeys {
		c = expandItemKey(c)
	}

	data, err = json.Marshal(&c)
	if err != nil {
		return nil, errors.Wrapf(err, "converting YAML to JSON failed")
	}
	return data, nil
}

func expandItemKey(c interface{}) interface{} {
	if c == nil {
		return c
	}

	// c is an array
	if cArr, isArray := c.([]interface{}); isArray {
		var arr []interface{}
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
				panic(errors.Wrapf(err, "unexpected error while setting key value (key: %s)", k))
			}
		}
		return expanded
	}
	return c // c is something else, e.g. string, int, etc.
}

func adjustErrorDescription(description string) string {
	if description == "Does not match format '"+relativePathFormat+"'" {
		return fmt.Sprintf("relative path is invalid, target doesn't exist or it exceeds the file size limit")
	} else if description == "Does not match format '"+dataStreamNameFormat+"'" {
		return "data stream doesn't exist"
	}
	return description
}
