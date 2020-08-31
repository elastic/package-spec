package validator

import (
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/pkg/errors"
	"github.com/xeipuuv/gojsonreference"
	"github.com/xeipuuv/gojsonschema"
	"gopkg.in/yaml.v3"
)

type yamlReferenceLoader struct {
	fs     http.FileSystem
	source string
}

var _ gojsonschema.JSONLoader = new(yamlReferenceLoader)

func (l *yamlReferenceLoader) JsonSource() interface{} {
	return l.source
}

func (l *yamlReferenceLoader) LoadJSON() (interface{}, error) {
	parsed, err := url.Parse(l.source)
	if err != nil {
		return nil, errors.Wrapf(err, "parsing source failed (source: %s)", l.source)
	}
	resourcePath := parsed.Path

	itemSchemaFile, err := l.fs.Open(resourcePath)
	if err != nil {
		return nil, errors.Wrapf(err, "opening schema file failed (path: %s)", resourcePath)
	}
	defer itemSchemaFile.Close()

	itemSchemaData, err := ioutil.ReadAll(itemSchemaFile)
	if err != nil {
		return nil, errors.Wrap(err, "reading schema file failed")
	}

	if len(itemSchemaData) == 0 {
		return nil, errors.New("schema file is empty")
	}

	var schema itemSchemaSpec
	err = yaml.Unmarshal(itemSchemaData, &schema)
	if err != nil {
		return nil, errors.Wrapf(err, "schema unmarshalling failed (path: %s)", l.source)
	}
	return schema.Spec, nil
}

func (l *yamlReferenceLoader) JsonReference() (gojsonreference.JsonReference, error) {
	return gojsonreference.NewJsonReference(l.JsonSource().(string))
}

func (l *yamlReferenceLoader) LoaderFactory() gojsonschema.JSONLoaderFactory {
	return &fileSystemYAMLLoaderFactory{
		fs: l.fs,
	}
}

func newReferenceLoaderFileSystem(source string, fs http.FileSystem) gojsonschema.JSONLoader {
	return &yamlReferenceLoader{
		fs:     fs,
		source: source,
	}
}

type fileSystemYAMLLoaderFactory struct {
	fs http.FileSystem
}

var _ gojsonschema.JSONLoaderFactory = new(fileSystemYAMLLoaderFactory)

func (f *fileSystemYAMLLoaderFactory) New(source string) gojsonschema.JSONLoader {
	return &yamlReferenceLoader{
		fs:     f.fs,
		source: source,
	}
}
