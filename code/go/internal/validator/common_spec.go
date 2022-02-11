// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package validator

import (
	"github.com/creasty/defaults"
	"github.com/pkg/errors"

	"github.com/elastic/package-spec/code/go/internal/spectypes"
)

type commonSpec struct {
	AdditionalContents bool               `yaml:"additionalContents"`
	Contents           []folderItemSpec   `yaml:"contents"`
	DevelopmentFolder  bool               `yaml:"developmentFolder"`
	MaxSize            spectypes.FileSize `yaml:"maxSize"`
	MaxContents        int                `yaml:"maxContents"`
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
