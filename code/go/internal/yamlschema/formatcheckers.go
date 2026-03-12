// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package yamlschema

import (
	"fmt"
	"io/fs"
	"path"

	jsonschema "github.com/santhosh-tekuri/jsonschema/v6"

	"github.com/elastic/package-spec/v3/code/go/internal/spectypes"
)

const (
	// relativePathFormat is the format identifier for fields whose value is a relative
	// filesystem path. The checker verifies the path exists in the package and does not
	// exceed the configured size limit.
	relativePathFormat = "relative-path"

	// dataStreamNameFormat is the format identifier for fields whose value is a data
	// stream name. The checker verifies a matching folder exists under data_stream/.
	dataStreamNameFormat = "data-stream-name"
)

// validationState holds the per-validation context used by format checker closures.
// It is populated before each validation call and cleared afterwards.
// Access is serialized by FileSchema.mu.
type validationState struct {
	fsys        fs.FS
	currentPath string
	sizeLimit   spectypes.FileSize
}

// newFormatCheckers returns Format objects for relative-path and data-stream-name
// validation. The closures capture state, which must be set before each validation call.
func newFormatCheckers(state *validationState) []*jsonschema.Format {
	return []*jsonschema.Format{
		{
			Name: relativePathFormat,
			Validate: func(v any) error {
				asString, ok := v.(string)
				if !ok || state.fsys == nil {
					return nil
				}
				return checkRelativePath(state.fsys, state.currentPath, asString, state.sizeLimit)
			},
		},
		{
			Name: dataStreamNameFormat,
			Validate: func(v any) error {
				asString, ok := v.(string)
				if !ok || state.fsys == nil {
					return nil
				}
				p := path.Join(state.currentPath, "data_stream")
				info, err := fs.Stat(state.fsys, path.Join(p, asString))
				if err != nil || !info.IsDir() {
					return fmt.Errorf("data stream doesn't exist")
				}
				return nil
			},
		},
	}
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
