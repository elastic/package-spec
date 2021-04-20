package pkgpath

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/PaesslerAG/jsonpath"
	"github.com/joeshaw/multierror"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

type File struct {
	os.FileInfo
}

func Files(glob string) ([]File, error) {
	paths, err := filepath.Glob(glob)
	if err != nil {
		return nil, err
	}

	var errs multierror.Errors
	var files = make([]File, 0)
	for _, path := range paths {
		info, err := os.Stat(path)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		file := File{info}
		files = append(files, file)
	}

	return files, errs.Err()
}

func (f File) Values(path string) (interface{}, error) {
	fileName := f.Name()
	fileExt := filepath.Ext(fileName)

	if fileExt != "json" && fileExt != "yaml" && fileExt != "yml" {
		return nil, fmt.Errorf("cannot extract values from file type = %s", fileExt)
	}

	contents, err := ioutil.ReadFile(fileName)
	if err != nil {
		return nil, errors.Wrap(err, "reading file content failed")
	}

	var v interface{}
	if fileExt == "yaml" || fileExt == "yml" {
		if err := yaml.Unmarshal(contents, &v); err != nil {
			return nil, errors.Wrapf(err, "unmarshalling YAML file failed (path: %s)", fileName)
		}
	} else if fileExt == "json" {
		if err := json.Unmarshal(contents, &v); err != nil {
			return nil, errors.Wrapf(err, "unmarshalling JSON file failed (path: %s)", fileName)
		}
	}

	return jsonpath.Get(path, v)
}
