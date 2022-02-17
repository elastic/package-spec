// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package validator

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"

	"github.com/elastic/package-spec/code/go/internal/spectypes"
)

const (
	maxConfigurationFileSize = 5 * spectypes.MegaByte
)

func validateContentTypeSize(path string, contentType spectypes.ContentType) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	size := spectypes.FileSize(info.Size())
	if size <= 0 {
		return errors.New("file is empty, but media type is defined")
	}

	var maxSize spectypes.FileSize
	switch contentType.MediaType {
	case "application/x-yaml":
		maxSize = maxConfigurationFileSize
	}
	if maxSize > 0 && size > maxSize {
		return errors.Errorf("file size (%s) is bigger than expected (%s)", size, maxSize)
	}
	return nil
}

func validateMaxSize(path string, maxSize spectypes.FileSize) error {
	if maxSize <= 0 {
		return nil
	}

	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	size := spectypes.FileSize(info.Size())
	if size > maxSize {
		return errors.Errorf("file size (%s) is bigger than expected (%s)", size, maxSize)
	}
	return nil
}

func processContentTypeData(data []byte, contentType spectypes.ContentType) ([]byte, error) {
	switch contentType.MediaType {
	case "application/x-yaml":
		// TODO Determine if special handling of `---` is required (issue: https://github.com/elastic/package-spec/pull/54)
		v, _ := contentType.Params["require-document-dashes"]
		if v == "true" && !bytes.HasPrefix(data, []byte("---\n")) {
			return nil, errors.New("document dashes are required (start the document with '---')")
		}

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
	case "application/json": // no need to convert the item content
	case "text/markdown": // text/markdown can't be transformed into JSON format
	case "text/plain": // text/plain should be left as-is
	default:
		return nil, fmt.Errorf("unsupported media type (%s)", contentType)
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
