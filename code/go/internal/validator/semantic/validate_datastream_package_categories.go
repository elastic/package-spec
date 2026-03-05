// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package semantic

import (
	"fmt"
	"io"
	"net/http"
	"path"
	"slices"
	"sort"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/elastic/package-spec/v3/code/go/internal/fspath"
	"github.com/elastic/package-spec/v3/code/go/pkg/specerrors"
)

const packageRegistryCategoriesURL = "https://raw.githubusercontent.com/elastic/package-registry/main/categories/categories.yml"

type registryCategories struct {
	Categories map[string]struct {
		Title         string `yaml:"title"`
		Subcategories map[string]struct {
			Title string `yaml:"title"`
		} `yaml:"subcategories"`
	} `yaml:"categories"`
}

func fetchRegistryParentCategories() (map[string]struct{}, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(packageRegistryCategoriesURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch categories from package registry: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read categories response: %w", err)
	}

	var rc registryCategories
	if err := yaml.Unmarshal(body, &rc); err != nil {
		return nil, fmt.Errorf("failed to parse categories YAML: %w", err)
	}

	parentCategories := make(map[string]struct{}, len(rc.Categories))
	for id := range rc.Categories {
		parentCategories[id] = struct{}{}
	}
	return parentCategories, nil
}

func readPackageManifestTypeAndCategories(fsys fspath.FS) (string, []string, error) {
	manifest, err := readManifest(fsys)
	if err != nil {
		return "", nil, err
	}

	typeVal, err := manifest.Values("$.type")
	if err != nil {
		return "", nil, fmt.Errorf("can't read manifest type: %w", err)
	}
	pkgType, ok := typeVal.(string)
	if !ok {
		return "", nil, fmt.Errorf("manifest type is not a string")
	}

	catsVal, err := manifest.Values("$.categories[*]")
	if err != nil {
		// categories field may be absent
		return pkgType, nil, nil
	}
	cats, err := toStringSlice(catsVal)
	if err != nil {
		return "", nil, fmt.Errorf("can't read manifest categories: %w", err)
	}
	return pkgType, cats, nil
}

// ValidateDatastreamPackageCategories validates that the package manifest
// categories include all parent-level categories present in any data stream
// manifest. Parent categories are determined by fetching the package registry
// categories.yml. Data stream manifests without a categories field are skipped.
func ValidateDatastreamPackageCategories(fsys fspath.FS) specerrors.ValidationErrors {
	manifestPath := "manifest.yml"
	pkgType, pkgCategories, err := readPackageManifestTypeAndCategories(fsys)
	if err != nil {
		return specerrors.ValidationErrors{
			specerrors.NewStructuredErrorf("file \"%s\" is invalid: %w", fsys.Path(manifestPath), err)}
	}

	if pkgType != packageTypeIntegration {
		return nil
	}

	dsCategories, err := readDataStreamManifestCategories(fsys)
	if err != nil {
		return specerrors.ValidationErrors{
			specerrors.NewStructuredErrorf("file \"%s\" is invalid: %w", fsys.Path(manifestPath), err)}
	}

	if len(dsCategories) == 0 {
		return nil
	}

	parentCategories, err := fetchRegistryParentCategories()
	if err != nil {
		return specerrors.ValidationErrors{
			specerrors.NewStructuredErrorf("file \"%s\" is invalid: failed to load registry categories: %w", fsys.Path(manifestPath), err)}
	}

	var errs specerrors.ValidationErrors
	for dsName, dsCats := range dsCategories {
		var missingCats []string
		for _, dsCat := range dsCats {
			if _, isParent := parentCategories[dsCat]; isParent && !slices.Contains(pkgCategories, dsCat) {
				missingCats = append(missingCats, dsCat)
			}
		}

		if len(missingCats) > 0 {
			sort.Strings(missingCats)
			dsManifestPath := path.Join("data_stream", dsName, "manifest.yml")
			errs = append(errs, specerrors.NewStructuredErrorf(
				"file \"%s\" is invalid: package manifest categories %v are missing parent categories %v from data stream \"%s\" (defined in \"%s\")",
				fsys.Path(manifestPath),
				pkgCategories,
				missingCats,
				dsName,
				fsys.Path(dsManifestPath),
			))
		}
	}

	return errs
}
