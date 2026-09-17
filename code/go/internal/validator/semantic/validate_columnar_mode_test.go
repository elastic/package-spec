// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package semantic

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/package-spec/v3/code/go/pkg/specerrors"
)

func TestDynamicIsFalse(t *testing.T) {
	cases := []struct {
		name  string
		value any
		want  bool
	}{
		{name: "bool false", value: false, want: true},
		{name: "bool true", value: true, want: false},
		{name: "string false", value: "false", want: true},
		{name: "string true", value: "true", want: false},
		{name: "string strict", value: "strict", want: false},
		{name: "nil", value: nil, want: false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.want, dynamicIsFalse(c.value))
		})
	}
}

func TestIsColumnarIndexMode(t *testing.T) {
	cases := []struct {
		mode string
		want bool
	}{
		{"logsdb_columnar", true},
		{"columnar", true},
		{"logsdb", false},
		{"time_series", false},
		{"", false},
	}
	for _, c := range cases {
		t.Run(c.mode, func(t *testing.T) {
			switch c.mode {
			case "logsdb_columnar", "columnar":
				assert.True(t, c.want)
			default:
				assert.False(t, c.want)
			}
		})
	}
}

func TestCheckColumnarField(t *testing.T) {
	meta := fieldFileMetadata{filePath: "fields.yml", fullFilePath: "/pkg/fields.yml"}

	cases := []struct {
		title    string
		f        field
		wantErrs bool
		wantCode string
	}{
		{
			title: "clean field passes",
			f:     field{Name: "host.name", Type: "keyword"},
		},
		{
			title:    "doc_values false is a hard error",
			f:        field{Name: "message", Type: "keyword", DocValues: boolPtr(false)},
			wantErrs: true,
			wantCode: specerrors.UnassignedCode,
		},
		{
			title:    "runtime field is a hard error",
			f:        field{Name: "computed", Type: "keyword", Runtime: runtimeField{enabled: true}},
			wantErrs: true,
			wantCode: specerrors.UnassignedCode,
		},
		{
			title:    "nested type is a filterable warning",
			f:        field{Name: "events", Type: "nested"},
			wantErrs: true,
			wantCode: specerrors.CodeColumnarNestedField,
		},
		{
			title:    "enabled false is a filterable warning",
			f:        field{Name: "metadata", Type: "object", Enabled: boolPtr(false)},
			wantErrs: true,
			wantCode: specerrors.CodeColumnarEnabledFalse,
		},
		{
			title:    "dynamic false on field is a filterable warning",
			f:        field{Name: "extra", Type: "object", Dynamic: false},
			wantErrs: true,
			wantCode: specerrors.CodeColumnarDynamicFalse,
		},
		{
			title:    "dynamic string false on field is a filterable warning",
			f:        field{Name: "extra", Type: "object", Dynamic: "false"},
			wantErrs: true,
			wantCode: specerrors.CodeColumnarDynamicFalse,
		},
		{
			title: "doc_values true is fine",
			f:     field{Name: "host.id", Type: "keyword", DocValues: boolPtr(true)},
		},
		{
			title: "enabled true is fine",
			f:     field{Name: "obj", Type: "object", Enabled: boolPtr(true)},
		},
	}

	for _, c := range cases {
		t.Run(c.title, func(t *testing.T) {
			errs := checkColumnarField(meta, c.f)
			if !c.wantErrs {
				assert.Empty(t, errs)
				return
			}
			assert.NotEmpty(t, errs)
			if c.wantCode != "" {
				assert.Equal(t, c.wantCode, errs[0].Code())
			}
		})
	}
}
