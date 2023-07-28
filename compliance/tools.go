// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build tools

package main

import (
	_ "github.com/elastic/elastic-package"
	_ "github.com/elastic/go-licenser"
	_ "golang.org/x/lint"
	_ "gotest.tools/gotestsum"
)
