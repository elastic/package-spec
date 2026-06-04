// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package semantic

import (
	"fmt"
	"path"
	"slices"
	"sort"

	"gopkg.in/yaml.v3"

	"github.com/elastic/package-spec/v3/code/go/internal/fspath"
	"github.com/elastic/package-spec/v3/code/go/internal/pkgpath"
	"github.com/elastic/package-spec/v3/code/go/pkg/specerrors"
)

type policyTemplateWithCategories struct {
	Name        string   `yaml:"name"`
	DataStreams []string `yaml:"data_streams"`
	Categories  []string `yaml:"categories"`
}

type packageManifestWithCategories struct {
	Type            string                         `yaml:"type"`
	PolicyTemplates []policyTemplateWithCategories `yaml:"policy_templates"`
}

func readPackageManifestPolicyTemplates(fsys fspath.FS) (string, []policyTemplateWithCategories, error) {
	manifest, err := readManifest(fsys)
	if err != nil {
		return "", nil, err
	}

	typeVal, err := manifest.Values("$.type")
	if err != nil {
		return "", nil, err
	}
	pkgType, ok := typeVal.(string)
	if !ok {
		return "", nil, fmt.Errorf("manifest type is not a string: %v", typeVal)
	}

	data, err := manifest.ReadAll()
	if err != nil {
		return "", nil, err
	}
	var pkg packageManifestWithCategories
	if err := yaml.Unmarshal(data, &pkg); err != nil {
		return "", nil, err
	}
	return pkgType, pkg.PolicyTemplates, nil
}

// ValidatePolicyTemplateDatastreamCategories validates that when a policy template
// entry in the package manifest.yml defines categories, each referenced data stream's
// manifest.yml categories include all of the policy template's categories. Data streams
// may declare additional categories beyond what the policy template specifies.
// Data stream manifests without a categories field are skipped.
func ValidatePolicyTemplateDatastreamCategories(fsys fspath.FS) specerrors.ValidationErrors {
	var errs specerrors.ValidationErrors

	manifestPath := "manifest.yml"
	pkgType, policyTemplates, err := readPackageManifestPolicyTemplates(fsys)
	if err != nil {
		return specerrors.ValidationErrors{
			specerrors.NewStructuredErrorf("file \"%s\" is invalid: %w", fsys.Path(manifestPath), err)}
	}

	// only validate integration type packages
	if pkgType != integrationPackageType {
		return nil
	}

	dsCategories, err := readDataStreamManifestCategories(fsys)
	if err != nil {
		return specerrors.ValidationErrors{
			specerrors.NewStructuredErrorf("file \"%s\" is invalid: %w", fsys.Path(manifestPath), err)}
	}

	for _, pt := range policyTemplates {
		// skip policy templates that don't define both categories and data_streams
		if len(pt.Categories) == 0 || len(pt.DataStreams) == 0 {
			continue
		}

		for _, dsName := range pt.DataStreams {
			dsCats, hasCats := dsCategories[dsName]
			if !hasCats {
				// data stream manifest has no categories field — nothing to validate
				continue
			}

			missing := missingCategories(pt.Categories, dsCats)
			if len(missing) > 0 {
				dsManifestPath := path.Join("data_stream", dsName, "manifest.yml")
				errs = append(errs, specerrors.NewStructuredErrorf(
					"file \"%s\" is invalid: data stream \"%s\" manifest categories %v are missing policy template \"%s\" categories %v (defined in \"%s\")",
					fsys.Path(manifestPath),
					dsName,
					dsCats,
					pt.Name,
					missing,
					fsys.Path(dsManifestPath),
				))
			}
		}
	}

	return errs
}

// readDataStreamManifestCategories reads the categories field from every
// data_stream/*/manifest.yml and returns a map of data stream name to its categories.
// Data streams without a categories field are omitted from the map.
func readDataStreamManifestCategories(fsys fspath.FS) (map[string][]string, error) {
	files, err := pkgpath.Files(fsys, "data_stream/*/manifest.yml")
	if err != nil {
		return nil, err
	}

	result := make(map[string][]string)
	for _, f := range files {
		catsVal, err := f.Values("$.categories[*]")
		if err != nil {
			continue
		}
		cats, err := toStringSlice(catsVal)
		if err != nil {
			return nil, fmt.Errorf("can't read categories from %s: %w", f.Path(), err)
		}
		if len(cats) == 0 {
			continue
		}
		// path is data_stream/{name}/manifest.yml — extract the name component
		dsName := path.Base(path.Dir(f.Path()))
		result[dsName] = cats
	}

	return result, nil
}

// missingCategories returns the categories present in want but absent from have.
// The result is sorted for deterministic error output.
func missingCategories(want, have []string) []string {
	var missing []string
	for _, c := range want {
		if !slices.Contains(have, c) {
			missing = append(missing, c)
		}
	}
	sort.Strings(missing)
	return missing
}
