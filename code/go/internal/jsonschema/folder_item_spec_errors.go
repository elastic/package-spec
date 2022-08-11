// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package jsonschema

import (
	"fmt"
)

func adjustErrorDescription(description string) string {
	if description == "Does not match format '"+relativePathFormat+"'" {
		return fmt.Sprintf("relative path is invalid, target doesn't exist or it exceeds the file size limit")
	} else if description == "Does not match format '"+dataStreamNameFormat+"'" {
		return "data stream doesn't exist"
	}
	return description
}
