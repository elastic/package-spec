// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build tools

package tools

import (
	_ "github.com/boumenot/gocover-cobertura"
	_ "golang.org/x/lint/golint"
	_ "gotest.tools/gotestsum"
	_ "honnef.co/go/tools/cmd/staticcheck"

	_ "github.com/elastic/go-licenser"
)
