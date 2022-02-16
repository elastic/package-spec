// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build go1.18

package spectypes

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func FuzzContentTypeMarshalling(f *testing.F) {
	f.Add(`"application/json"`)
	f.Add(`"application/x-yaml; require-document-dashes=true"`)
	f.Add(`"application/x-yaml; require-document-dashes=true; charset=utf-8"`)

	f.Fuzz(func(t *testing.T, contentType string) {
		t.Log("original: " + contentType)

		var first, second ContentType
		err := json.Unmarshal([]byte(contentType), &first)
		if err != nil {
			return
		}

		t.Logf("first: (%s)", first)
		d, err := json.Marshal(first)
		require.NoError(t, err)

		err = json.Unmarshal(d, &second)
		require.NoError(t, err)

		t.Logf("second: (%s)", second)
		require.Equal(t, first.MediaType, second.MediaType)
		require.EqualValues(t, first.Params, second.Params)
	})
}

func FuzzFileSizeMarshalling(f *testing.F) {
	f.Add(`0`)
	f.Add(`1024`)
	f.Add(`"0B"`)
	f.Add(`"1KB"`)
	f.Add(`"1025B"`)
	f.Add(`"5MB"`)

	f.Fuzz(func(t *testing.T, fileSize string) {
		t.Log("original: " + fileSize)

		var first, second FileSize
		err := json.Unmarshal([]byte(fileSize), &first)
		if err != nil {
			return
		}

		t.Logf("first: (%s)", first)
		d, err := json.Marshal(first)
		require.NoError(t, err)

		err = json.Unmarshal(d, &second)
		require.NoError(t, err)

		t.Logf("second: (%s)", second)
		require.Equal(t, first, second)
	})
}
