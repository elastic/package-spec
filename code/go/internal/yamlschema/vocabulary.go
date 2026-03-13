// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package yamlschema

import (
	"fmt"
	"io/fs"
	"path"

	jsonschema "github.com/santhosh-tekuri/jsonschema/v6"
	"golang.org/x/text/message"

	"github.com/elastic/package-spec/v3/code/go/internal/spectypes"
)

const (
	// elasticRelativePathKeyword is a custom JSON Schema keyword that validates
	// a string field as a relative filesystem path. The validator checks that the
	// path exists in the package and does not exceed the configured size limit.
	elasticRelativePathKeyword = "x-elastic-relative-path"

	// elasticDataStreamNameKeyword is a custom JSON Schema keyword that validates
	// a string field as a data stream name. The validator checks that a matching
	// directory exists under data_stream/.
	elasticDataStreamNameKeyword = "x-elastic-data-stream-name"

	// elasticVocabularyURL is the identifier for the Elastic package-spec vocabulary.
	elasticVocabularyURL = "https://github.com/elastic/package-spec"
)

// elasticVocabulary owns the per-validation context and exposes the jsonschema.Vocabulary
// to register with the compiler. SetContext/ClearContext are called by FileSchema.Validate
// to make filesystem information available to the SchemaExt closures.
type elasticVocabulary struct {
	fsys        fs.FS
	currentPath string
	sizeLimit   spectypes.FileSize
	vocab       *jsonschema.Vocabulary
}

func newElasticVocabulary() *elasticVocabulary {
	ev := &elasticVocabulary{}
	ev.vocab = &jsonschema.Vocabulary{
		URL: elasticVocabularyURL,
		Compile: func(_ *jsonschema.CompilerContext, obj map[string]any) (jsonschema.SchemaExt, error) {
			if _, ok := obj[elasticRelativePathKeyword]; ok {
				return &relativePathExt{vocabulary: ev}, nil
			}
			if _, ok := obj[elasticDataStreamNameKeyword]; ok {
				return &dataStreamNameExt{vocabulary: ev}, nil
			}
			return nil, nil
		},
	}
	return ev
}

func (ev *elasticVocabulary) setContext(fsys fs.FS, currentPath string, sizeLimit spectypes.FileSize) {
	ev.fsys = fsys
	ev.currentPath = currentPath
	ev.sizeLimit = sizeLimit
}

func (ev *elasticVocabulary) clearContext() {
	ev.fsys = nil
}

type relativePathExt struct {
	vocabulary *elasticVocabulary
}

func (e *relativePathExt) Validate(ctx *jsonschema.ValidatorContext, v any) {
	if e.vocabulary.fsys == nil {
		panic("elasticVocabulary: setContext must be called before Validate")
	}
	str, ok := v.(string)
	if !ok {
		return
	}
	if err := checkRelativePath(e.vocabulary.fsys, e.vocabulary.currentPath, str, e.vocabulary.sizeLimit); err != nil {
		ctx.AddError(&relativePathErrorKind{Value: str, Err: err})
	}
}

type relativePathErrorKind struct {
	Value string
	Err   error
}

func (*relativePathErrorKind) KeywordPath() []string {
	return []string{elasticRelativePathKeyword}
}

func (k *relativePathErrorKind) LocalizedString(p *message.Printer) string {
	return p.Sprintf("%q is not a valid relative path: %v", k.Value, k.Err)
}

type dataStreamNameExt struct {
	vocabulary *elasticVocabulary
}

func (e *dataStreamNameExt) Validate(ctx *jsonschema.ValidatorContext, v any) {
	if e.vocabulary.fsys == nil {
		panic("elasticVocabulary: SetContext must be called before Validate")
	}
	str, ok := v.(string)
	if !ok {
		return
	}
	p := path.Join(e.vocabulary.currentPath, "data_stream")
	info, err := fs.Stat(e.vocabulary.fsys, path.Join(p, str))
	if err != nil || !info.IsDir() {
		ctx.AddError(&dataStreamNameErrorKind{Value: str})
	}
}

type dataStreamNameErrorKind struct {
	Value string
}

func (*dataStreamNameErrorKind) KeywordPath() []string {
	return []string{elasticDataStreamNameKeyword}
}

func (k *dataStreamNameErrorKind) LocalizedString(p *message.Printer) string {
	return p.Sprintf("%q is not a valid data stream name: data stream doesn't exist", k.Value)
}

func checkRelativePath(fsys fs.FS, base, rel string, sizeLimit spectypes.FileSize) error {
	p := path.Join(base, rel)
	info, err := fs.Stat(fsys, p)
	if err != nil {
		return fmt.Errorf("relative path is invalid, target doesn't exist or it exceeds the file size limit")
	}
	if sizeLimit > 0 && spectypes.FileSize(info.Size()) > sizeLimit {
		return fmt.Errorf("relative path is invalid, target doesn't exist or it exceeds the file size limit")
	}
	return nil
}
