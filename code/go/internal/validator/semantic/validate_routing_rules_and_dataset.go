// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package semantic

import (
	"fmt"
	"io/fs"
	"path"

	"gopkg.in/yaml.v3"

	ve "github.com/elastic/package-spec/v2/code/go/internal/errors"
	"github.com/elastic/package-spec/v2/code/go/internal/fspath"
	"github.com/elastic/package-spec/v2/code/go/internal/pkgpath"
	pve "github.com/elastic/package-spec/v2/code/go/pkg/errors"
)

// ValidateRoutingRulesAndDataset returns validation errors if there are routing rules defined in any dataStream
// but that dataStream does not defines "dataset" field.
func ValidateRoutingRulesAndDataset(fsys fspath.FS) pve.ValidationErrors {
	dataStreams, err := listDataStreams(fsys)
	if err != nil {
		return pve.ValidationErrors{ve.NewStructuredError(err, "data_stream", "", ve.Critical)}
	}

	var errs pve.ValidationErrors
	for _, dataStream := range dataStreams {
		anyRoutingRules, err := anyRoutingRulesInDataStream(fsys, dataStream)
		if !anyRoutingRules {
			continue
		}
		err = validateDatasetInDataStream(fsys, dataStream)
		if err != nil {
			vError := ve.NewStructuredError(
				fmt.Errorf("routing rules defined in data stream %q but dataset field is missing: %w", dataStream, err),
				routingRulesPath(dataStream),
				"",
				ve.Critical,
			)
			errs.Append(pve.ValidationErrors{vError})
		}
	}
	return errs
}

func validateDatasetInDataStream(fsys fspath.FS, dataStream string) error {
	manifestPath := path.Join("data_stream", dataStream, "manifest.yml")
	d, err := fs.ReadFile(fsys, manifestPath)
	if err != nil {
		return fmt.Errorf("failed to read data stream manifest in %q: %w", fsys.Path(manifestPath), err)
	}

	var manifest struct {
		Dataset string `yaml:"dataset,omitempty"`
	}
	err = yaml.Unmarshal(d, &manifest)
	if err != nil {
		return fmt.Errorf("failed to parse data stream manifest in %q: %w", fsys.Path(manifestPath), err)
	}

	if manifest.Dataset == "" {
		return fmt.Errorf("dataset field is required in data stream %q", dataStream)
	}
	return nil
}

func routingRulesPath(dataStream string) string {
	return path.Join("data_stream", dataStream, "routing_rules.yml")
}

func anyRoutingRulesInDataStream(fsys fspath.FS, dataStream string) (bool, error) {
	routingRulesPath := routingRulesPath(dataStream)
	f, err := pkgpath.Files(fsys, routingRulesPath)
	if err != nil {
		return false, nil
	}

	if len(f) == 0 {
		return false, nil
	}

	if len(f) != 1 {
		return false, fmt.Errorf("single routing rules expected")
	}

	vals, err := f[0].Values("$[*]")
	if err != nil {
		return false, fmt.Errorf("can't read routing_rules: %w", err)
	}

	rules, ok := vals.([]interface{})
	if !ok {
		return false, fmt.Errorf("routing rules conversion error")
	}
	if len(rules) > 0 {
		return true, nil
	}
	return false, nil
}
