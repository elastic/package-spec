// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package validator

import (
	"fmt"

	"github.com/elastic/package-spec/code/go/internal/validator/semantic"
)

func adjustErrorDescription(description string) string {
	if description == "Does not match format '"+semantic.RelativePathFormat+"'" {
		return fmt.Sprintf("relative path is invalid, target doesn't exist or is bigger than %s", semantic.RelativePathFileMaxSize)
	} else if description == "Does not match format '"+semantic.DataStreamNameFormat+"'" {
		return "data stream doesn't exist"
	}
	return description
}
