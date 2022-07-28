// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package spec

import (
	"embed"
	"io/fs"
)

//go:embed spec spec/integration/_dev spec/integration/data_stream/_dev spec/input
var content embed.FS

// FS returns an io/fs.FS for accessing the "package-spec/spec" contents.
func FS() fs.FS {
	fs, err := fs.Sub(content, "spec")
	if err != nil {
		panic(err)
	}
	return fs
}
