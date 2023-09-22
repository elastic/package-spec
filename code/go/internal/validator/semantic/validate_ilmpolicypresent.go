// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package semantic

import (
	"fmt"
	"io/fs"
	"path"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/elastic/package-spec/v2/code/go/internal/fspath"
	ve "github.com/elastic/package-spec/v2/code/go/pkg/errors"
)

// ValidateILMPolicyPresent produces an error if the indicated ILM policy
// is not defined in the data stream.
func ValidateILMPolicyPresent(fsys fspath.FS) ve.ValidationErrors {
	dataStreams, err := listDataStreams(fsys)
	if err != nil {
		return ve.ValidationErrors{ve.NewStructuredError(err, ve.UnassignedCode)}
	}

	var errs ve.ValidationErrors
	for _, dataStream := range dataStreams {
		err = validateILMPolicyInDataStream(fsys, dataStream)
		if err != nil {
			errs = append(errs, ve.NewStructuredError(err, ve.UnassignedCode))
		}
	}
	return errs
}

func validateILMPolicyInDataStream(fsys fspath.FS, dataStream string) error {
	dsType, ilmPolicy, err := readILMPolicyInfoInDataStream(fsys, dataStream)
	if err != nil {
		return err
	}

	if ilmPolicy == "" {
		return nil
	}

	packageName, err := readPackageName(fsys)
	if err != nil {
		return err
	}

	manifestPath := path.Join("data_stream", dataStream, "manifest.yml")
	policyPrefix := expectedILMPolicyPrefix(dsType, packageName, dataStream)
	if !strings.HasPrefix(ilmPolicy, policyPrefix) {
		return fmt.Errorf("file \"%s\" is invalid: field ilm_policy must start with %q, found \"%s\"", fsys.Path(manifestPath), policyPrefix, ilmPolicy)
	}

	ilmFileName := ilmPolicy[len(policyPrefix):] + ".json"
	ilmFilePath := path.Join("data_stream", dataStream, "elasticsearch", "ilm", ilmFileName)
	_, err = fs.Stat(fsys, ilmFilePath)
	if err != nil {
		return fmt.Errorf("file \"%s\" is invalid: field ilm_policy: ILM policy %q not found in package, expected definition in \"%s\"", fsys.Path(manifestPath), ilmPolicy, fsys.Path(ilmFilePath))
	}

	return nil
}

func readILMPolicyInfoInDataStream(fsys fspath.FS, dataStream string) (dsType string, ilmPolicy string, err error) {
	manifestPath := path.Join("data_stream", dataStream, "manifest.yml")

	d, err := fs.ReadFile(fsys, manifestPath)
	if err != nil {
		return "", "", fmt.Errorf("failed to read data stream manifest in %q: %w", fsys.Path(manifestPath), err)
	}

	var manifest struct {
		Type      string `yaml:"type"`
		ILMPolicy string `yaml:"ilm_policy"`
	}
	err = yaml.Unmarshal(d, &manifest)
	if err != nil {
		return "", "", fmt.Errorf("failed to parse data stream manifest in %q: %w", fsys.Path(manifestPath), err)
	}

	return manifest.Type, manifest.ILMPolicy, nil
}

func readPackageName(fsys fspath.FS) (string, error) {
	manifestPath := "manifest.yml"

	d, err := fs.ReadFile(fsys, manifestPath)
	if err != nil {
		return "", fmt.Errorf("failed to manifest in %q: %w", fsys.Path(manifestPath), err)
	}

	var manifest struct {
		Name string `yaml:"name"`
	}
	err = yaml.Unmarshal(d, &manifest)
	if err != nil {
		return "", fmt.Errorf("failed to parse manifest in %q: %w", fsys.Path(manifestPath), err)
	}

	return manifest.Name, nil
}

func expectedILMPolicyPrefix(dsType, packageName, dataStream string) string {
	return fmt.Sprintf("%s-%s.%s-", dsType, packageName, dataStream)
}
