// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package pkgpath

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/PaesslerAG/jsonpath"
	"github.com/joeshaw/multierror"
	"github.com/pkg/errors"

	"github.com/elastic/go-ucfg/json"
	"github.com/elastic/go-ucfg/yaml"
	"github.com/elastic/package-spec/v2/code/go/internal/fspath"
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
		return nil, errors.Wrap(err, "reading file content failed")
	}

	v := make(map[string]interface{})
	if fileExt == "yaml" || fileExt == "yml" {
		config, err := yaml.NewConfig(contents)
		if err != nil {
			return nil, errors.Wrapf(err, "parsing yaml file failed (path: %s)", f.fsys.Path(fileName))
		}

		if err = config.Unpack(&v); err != nil {
			return nil, errors.Wrap(err, "can't unpack file (path: %s)", f.fsys.Path(fileName))
		}
	} else if fileExt == "json" {
		config, err := json.NewConfig(contents)
		if err != nil {
			return nil, errors.Wrapf(err, "parsing json file failed (path: %s)", f.fsys.Path(fileName))
		}
		config.Unpack(&v)
	}

	return jsonpath.Get(path, v)
}

// ValuesArray Same as values but returns an array of values
func (f File) ValuesArray(path string) ([]interface{}, error) {
	fileName := f.Name()
	fileExt := strings.TrimLeft(filepath.Ext(fileName), ".")

	if fileExt != "json" && fileExt != "yaml" && fileExt != "yml" {
		return nil, fmt.Errorf("cannot extract values from file type = %s", fileExt)
	}

	contents, err := fs.ReadFile(f.fsys, f.path)
	if err != nil {
		return nil, errors.Wrap(err, "reading file content failed")
	}

	var v []interface{}
	if fileExt == "yaml" || fileExt == "yml" {
		config, err := yaml.NewConfig(contents)
		if err != nil {
			return nil, errors.Wrapf(err, "parsing yaml file failed (path: %s)", f.fsys.Path(fileName))
		}

		config.Unpack(&v)
	} else if fileExt == "json" {
		config, err := json.NewConfig(contents)
		if err != nil {
			return nil, errors.Wrapf(err, "parsing json file failed (path: %s)", f.fsys.Path(fileName))
		}
		config.Unpack(&v)
	}

	pathVal, pathErr := jsonpath.Get(path, v)

	if pathErr != nil {
		return nil, pathErr
	}

	vals, ok := pathVal.([]interface{})

	if !ok {
		return nil, errors.New("conversion error")
	}

	return vals, nil
}

// Path returns the complete path to the file.
func (f File) Path() string {
	return f.path
}
