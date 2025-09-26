// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build tools

package tools

import (
	_ "github.com/boumenot/gocover-cobertura"
	_ "github.com/elastic/go-licenser"
	_ "golang.org/x/lint/golint"
	_ "golang.org/x/tools/cmd/goimports"
	_ "gotest.tools/gotestsum"
)
