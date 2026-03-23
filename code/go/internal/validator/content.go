// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package validator

import (
	"bufio"
	"errors"
	"fmt"
	"io/fs"

	"github.com/elastic/package-spec/v3/code/go/internal/spectypes"
)

func validateContentType(fsys fs.FS, path string, contentType spectypes.ContentType) error {
	switch contentType.MediaType {
	case "application/x-yaml":
		v := contentType.Params["require-document-dashes"]
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
	case "text/csv":
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

	scanner := bufio.NewScanner(f)
	// A small buffer should be enough to check if the document starts with three dashes.
	buf := make([]byte, 8)
	scanner.Buffer(buf, len(buf))
	scanner.Scan()
	if err := scanner.Err(); err != bufio.ErrTooLong && err != nil {
		return err
	}
	if scanner.Text() != "---" {
		return errors.New("document dashes are required (start the document with '---')")
	}

	return nil
}

func validateContentTypeSize(fsys fs.FS, path string, contentType spectypes.ContentType, limits spectypes.LimitsSpec) error {
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
		sizeLimit = limits.MaxConfigurationSize()
	}
	if sizeLimit > 0 && size > sizeLimit {
		return fmt.Errorf("file size (%s) is bigger than expected (%s)", size, sizeLimit)
	}
	return nil
}

func validateMaxSize(fsys fs.FS, path string, limits spectypes.LimitsSpec) error {
	if limits.MaxFileSize() == 0 {
		return nil
	}

	info, err := fs.Stat(fsys, path)
	if err != nil {
		return err
	}
	size := spectypes.FileSize(info.Size())
	if size > limits.MaxFileSize() {
		return fmt.Errorf("file size (%s) is bigger than expected (%s)", size, limits.MaxFileSize())
	}
	return nil
}
