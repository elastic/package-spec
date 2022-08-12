// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package specschema

import (
	"reflect"

	"github.com/creasty/defaults"
	"github.com/pkg/errors"

	"github.com/elastic/package-spec/code/go/internal/spectypes"
)

type commonSpec struct {
	AdditionalContents bool              `yaml:"additionalContents"`
	Contents           []*folderItemSpec `yaml:"contents"`
	DevelopmentFolder  bool              `yaml:"developmentFolder"`

	Limits commonSpecLimits `yaml:",inline"`

	// Release type of the spec: beta, ga.
	// Packages using beta features won't be able to go GA.
	// Default release: ga
	Release string `yaml:"release"`
}

type commonSpecLimits struct {
	// Limit to the total number of elements in a directory.
	TotalContentsLimit int `yaml:"totalContentsLimit"`

	// Limit to the total size of files in a directory.
	TotalSizeLimit spectypes.FileSize `yaml:"totalSizeLimit"`

	// Limit to individual files.
	SizeLimit spectypes.FileSize `yaml:"sizeLimit"`

	// Limit to individual configuration files (yaml files).
	ConfigurationSizeLimit spectypes.FileSize `yaml:"configurationSizeLimit"`

	// Limit to files referenced as relative paths (images).
	RelativePathSizeLimit spectypes.FileSize `yaml:"relativePathSizeLimit"`

	// Maximum number of fields per data stream, can only be set at the root level spec.
	FieldsPerDataStreamLimit int `yaml:"fieldsPerDataStreamLimit"`
}

func (l *commonSpecLimits) update(o commonSpecLimits) {
	target := reflect.ValueOf(l).Elem()
	source := reflect.ValueOf(&o).Elem()
	for i := 0; i < target.NumField(); i++ {
		field := target.Field(i)
		if field.IsZero() {
			field.Set(source.Field(i))
		}
	}
}

func setDefaultValues(spec *commonSpec) error {
	err := defaults.Set(spec)
	if err != nil {
		return errors.Wrap(err, "could not set default values")
	}

	if len(spec.Contents) == 0 {
		return nil
	}

	for i := range spec.Contents {
		err = setDefaultValues(&spec.Contents[i].commonSpec)
		if err != nil {
			return err
		}
	}

	return nil
}

func propagateContentLimits(spec *commonSpec) {
	for i := range spec.Contents {
		content := &spec.Contents[i].commonSpec
		content.Limits.update(spec.Limits)
		propagateContentLimits(content)
	}
}
