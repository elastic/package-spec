// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package pkgpath

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/PaesslerAG/jsonpath"
	"github.com/joeshaw/multierror"

	"gopkg.in/yaml.v3"

	"github.com/elastic/package-spec/v3/code/go/internal/fspath"
)

// File represents a file in the package.
type File struct {
	fsys fspath.FS
	path string
	os.FileInfo
}

// Files finds files for the given glob
func Files(fsys fspath.FS, glob string) ([]File, error) {
	paths, err := fs.Glob(fsys, glob)
	if err != nil {
		return nil, err
	}

	var errs multierror.Errors
	var files = make([]File, 0)
	for _, path := range paths {
		info, err := fs.Stat(fsys, path)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		file := File{fsys, path, info}
		files = append(files, file)
	}

	return files, errs.Err()
}

// Values returns values within the file matching the given path. Paths
// should be expressed using JSONPath syntax. This method is only supported
// for YAML and JSON files.
func (f File) Values(path string) (interface{}, error) {
	fileName := f.Name()
	fileExt := strings.TrimLeft(filepath.Ext(fileName), ".")

	if fileExt != "json" && fileExt != "yaml" && fileExt != "yml" {
		return nil, fmt.Errorf("cannot extract values from file type = %s", fileExt)
	}

	contents, err := fs.ReadFile(f.fsys, f.path)
	if err != nil {
		return nil, fmt.Errorf("reading file content failed: %w", err)
	}

	var v interface{}
	if fileExt == "yaml" || fileExt == "yml" {
		if err := yaml.Unmarshal(contents, &v); err != nil {
			return nil, fmt.Errorf("unmarshalling YAML file failed (path: %s): %w", f.fsys.Path(fileName), err)
		}
	} else if fileExt == "json" {
		if err := json.Unmarshal(contents, &v); err != nil {
			return nil, fmt.Errorf("unmarshalling JSON file failed (path: %s): %w", f.fsys.Path(fileName), err)
		}
	}

	return jsonpath.Get(path, v)
}

// Path returns the complete path to the file.
func (f File) Path() string {
	return f.path
}

// ReadAll reads and returns the entire contents of the file.
func (f File) ReadAll() ([]byte, error) {
	return fs.ReadFile(f.fsys, f.path)
}
