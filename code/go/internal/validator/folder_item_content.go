// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package validator

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"

	ve "github.com/elastic/package-spec/code/go/internal/errors"
	"github.com/elastic/package-spec/code/go/internal/spectypes"
)

func loadItemSchema(fsys fs.FS, path string, contentType *spectypes.ContentType) ([]byte, error) {
	data, err := fs.ReadFile(fsys, path)
	if err != nil {
		return nil, ve.ValidationErrors{errors.Wrap(err, "reading item file failed")}
	}
	if contentType != nil && contentType.MediaType == "application/x-yaml" {
		return convertYAMLToJSON(data)
	}
	return data, nil
}

func validateContentType(fsys fs.FS, path string, contentType spectypes.ContentType) error {
	switch contentType.MediaType {
	case "application/x-yaml":
		v, _ := contentType.Params["require-document-dashes"]
		requireDashes := (v == "true")
		if requireDashes {
			err := validateYAMLDashes(fsys, path)
			if err != nil {
				return err
			}
		}
	case "application/json":
	case "text/markdown":
	case "text/plain":
	default:
		return fmt.Errorf("unsupported media type (%s)", contentType)
	}
	return nil
}

func validateYAMLDashes(fsys fs.FS, path string) error {
	f, err := fsys.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	dashes := []byte("---\n")
	b := make([]byte, len(dashes))
	_, err = f.Read(b)
	if err != nil {
		return err
	}

	if !bytes.Equal(dashes, b) {
		return errors.New("document dashes are required (start the document with '---')")
	}
	return nil
}

func validateContentTypeSize(fsys fs.FS, path string, contentType spectypes.ContentType, limits commonSpecLimits) error {
	info, err := fs.Stat(fsys, path)
	if err != nil {
		return err
	}
	size := spectypes.FileSize(info.Size())
	if size <= 0 {
		return errors.New("file is empty, but media type is defined")
	}

	var sizeLimit spectypes.FileSize
	switch contentType.MediaType {
	case "application/x-yaml":
		sizeLimit = limits.ConfigurationSizeLimit
	}
	if sizeLimit > 0 && size > sizeLimit {
		return errors.Errorf("file size (%s) is bigger than expected (%s)", size, sizeLimit)
	}
	return nil
}

func validateMaxSize(fsys fs.FS, path string, limits commonSpecLimits) error {
	if limits.SizeLimit == 0 {
		return nil
	}

	info, err := fs.Stat(fsys, path)
	if err != nil {
		return err
	}
	size := spectypes.FileSize(info.Size())
	if size > limits.SizeLimit {
		return errors.Errorf("file size (%s) is bigger than expected (%s)", size, limits.SizeLimit)
	}
	return nil
}

func convertYAMLToJSON(data []byte) ([]byte, error) {
	var c interface{}
	err := yaml.Unmarshal(data, &c)
	if err != nil {
		return nil, errors.Wrapf(err, "unmarshalling YAML file failed")
	}
	c = expandItemKey(c)

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
