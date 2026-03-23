// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package semantic

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/elastic/package-spec/v3/code/go/internal/fspath"
)

func TestValidateDurationVar(t *testing.T) {
	tests := []struct {
		name     string
		variable durationVar
		wantErrs bool
	}{
		{
			name: "non-duration type",
			variable: durationVar{
				Name: "test_var",
				Type: "text",
			},
			wantErrs: false,
		},
		{
			name: "valid duration - no constraints",
			variable: durationVar{
				Name: "test_var",
				Type: "duration",
			},
			wantErrs: false,
		},
		{
			name: "valid duration - with default",
			variable: durationVar{
				Name:    "test_var",
				Type:    "duration",
				Default: "10s",
			},
			wantErrs: false,
		},
		{
			name: "valid duration - with min",
			variable: durationVar{
				Name:        "test_var",
				Type:        "duration",
				MinDuration: strPtr("5s"),
			},
			wantErrs: false,
		},
		{
			name: "valid duration - with max",
			variable: durationVar{
				Name:        "test_var",
				Type:        "duration",
				MaxDuration: strPtr("30s"),
			},
			wantErrs: false,
		},
		{
			name: "valid duration - with min and default",
			variable: durationVar{
				Name:        "test_var",
				Type:        "duration",
				MinDuration: strPtr("5s"),
				Default:     "10s",
			},
			wantErrs: false,
		},
		{
			name: "valid duration - with default and max",
			variable: durationVar{
				Name:        "test_var",
				Type:        "duration",
				Default:     "10s",
				MaxDuration: strPtr("30s"),
			},
			wantErrs: false,
		},
		{
			name: "valid duration - with min and max",
			variable: durationVar{
				Name:        "test_var",
				Type:        "duration",
				MinDuration: strPtr("5s"),
				MaxDuration: strPtr("30s"),
			},
			wantErrs: false,
		},
		{
			name: "valid duration - with min, default, and max",
			variable: durationVar{
				Name:        "test_var",
				Type:        "duration",
				MinDuration: strPtr("5s"),
				Default:     "10s",
				MaxDuration: strPtr("30s"),
			},
			wantErrs: false,
		},
		{
			name: "valid duration - with equal min and default",
			variable: durationVar{
				Name:        "test_var",
				Type:        "duration",
				MinDuration: strPtr("10s"),
				Default:     "10s",
			},
			wantErrs: false,
		},
		{
			name: "valid duration - with equal default and max",
			variable: durationVar{
				Name:        "test_var",
				Type:        "duration",
				Default:     "30s",
				MaxDuration: strPtr("30s"),
			},
			wantErrs: false,
		},
		{
			name: "valid duration - with equal min and max",
			variable: durationVar{
				Name:        "test_var",
				Type:        "duration",
				MinDuration: strPtr("10s"),
				MaxDuration: strPtr("10s"),
			},
			wantErrs: false,
		},
		{
			name: "valid duration - with all equal",
			variable: durationVar{
				Name:        "test_var",
				Type:        "duration",
				MinDuration: strPtr("10s"),
				Default:     "10s",
				MaxDuration: strPtr("10s"),
			},
			wantErrs: false,
		},
		{
			name: "invalid duration - negative min",
			variable: durationVar{
				Name:        "test_var",
				Type:        "duration",
				MinDuration: strPtr("-5s"),
			},
			wantErrs: true,
		},
		{
			name: "invalid duration - min > default",
			variable: durationVar{
				Name:        "test_var",
				Type:        "duration",
				MinDuration: strPtr("15s"),
				Default:     "10s",
			},
			wantErrs: true,
		},
		{
			name: "invalid duration - default > max",
			variable: durationVar{
				Name:        "test_var",
				Type:        "duration",
				Default:     "40s",
				MaxDuration: strPtr("30s"),
			},
			wantErrs: true,
		},
		{
			name: "invalid duration - min > max",
			variable: durationVar{
				Name:        "test_var",
				Type:        "duration",
				MinDuration: strPtr("40s"),
				MaxDuration: strPtr("30s"),
			},
			wantErrs: true,
		},
		{
			name: "invalid duration - multiple violations",
			variable: durationVar{
				Name:        "test_var",
				Type:        "duration",
				MinDuration: strPtr("40s"),
				Default:     "20s",
				MaxDuration: strPtr("30s"),
			},
			wantErrs: true, // min > default and min > max
		},
		{
			name: "invalid duration - unparseable min",
			variable: durationVar{
				Name:        "test_var",
				Type:        "duration",
				MinDuration: strPtr("invalid"),
			},
			wantErrs: true,
		},
		{
			name: "invalid duration - unparseable default",
			variable: durationVar{
				Name:    "test_var",
				Type:    "duration",
				Default: strPtr("invalid"),
			},
			wantErrs: true,
		},
		{
			name: "invalid duration - unparseable max",
			variable: durationVar{
				Name:        "test_var",
				Type:        "duration",
				MaxDuration: strPtr("invalid"),
			},
			wantErrs: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateDurationVar(tt.variable)
			if tt.wantErrs {
				if err == nil {
					t.Fatalf("Expected error, got nil")
				}
				t.Logf("Expected error: %v", err)
			} else if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
		})
	}
}

func strPtr(s string) *string {
	return &s
}

func TestValidateDurationVariables(t *testing.T) {
	const testManifest = `
vars:
  - name: interval
    type: duration
    min_duration: 5s
    max_duration: 30s
    default: 1m
  - name: foo
    type: text
    multi: true
    default:
      - bar
      - baz
policy_templates:
  - vars:
      - name: wait_time
        type: duration
        min_duration: 30s
        max_duration: 5s
    inputs:
      - vars:
          - name: period
            type: duration
            min_duration: -5s
            default: 10s
`

	const testDataStreamManifest = `
streams:
  - vars:
      - name: stream_interval
        type: duration
        max_duration: 1h1h
        default: 100ms
  - vars:
      - name: dwell_time
        type: duration
        min_duration: 50ms50ms
        max_duration: 1h
        default: 5ms
`

	d := t.TempDir()
	if err := os.WriteFile(filepath.Join(d, "manifest.yml"), []byte(testManifest[1:]), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(d, "data_stream/foo"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(d, "data_stream/foo/manifest.yml"), []byte(testDataStreamManifest[1:]), 0o600); err != nil {
		t.Fatal(err)
	}

	want := []string{
		`manifest.yml:2:5 error in variable "interval": default "1m" greater than max_duration "30s"`,
		`manifest.yml:15:9 error in variable "wait_time": min_duration "30s" greater than max_duration "5s"`,
		`manifest.yml:21:13 error in variable "period": negative min_duration value "-5s"`,
		filepath.Join("data_stream", "foo", "manifest.yml") + `:8:9 error in variable "dwell_time": min_duration "50ms50ms" greater than default "5ms"`,
	}

	errs := ValidateDurationVariables(fspath.DirFS(d))
	if len(errs) != len(want) {
		t.Fatalf("Expected %d errors, got %d", len(want), len(errs))
	}

	for i, err := range errs {
		got := err.Error()
		if !strings.Contains(got, want[i]) {
			t.Errorf("at index %d, want=`%s`, got=`%s`", i, want[i], got)
		}
	}
}
