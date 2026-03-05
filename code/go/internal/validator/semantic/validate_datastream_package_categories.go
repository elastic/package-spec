// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package semantic

import (
	"fmt"
	"io"
	"io/fs"
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

type packageManifestWithPackageCategories struct {
	Type       string   `yaml:"type"`
	Categories []string `yaml:"categories"`
}

// ValidateDatastreamPackageCategories validates that the package manifest
// categories include all parent-level categories present in any data stream
// manifest. Parent categories are determined by fetching the package registry
// categories.yml. Data stream manifests without a categories field are skipped.
func ValidateDatastreamPackageCategories(fsys fspath.FS) specerrors.ValidationErrors {
	manifestPath := "manifest.yml"
	data, err := fs.ReadFile(fsys, manifestPath)
	if err != nil {
		return specerrors.ValidationErrors{
			specerrors.NewStructuredErrorf("file \"%s\" is invalid: %w", fsys.Path(manifestPath), errFailedToReadManifest)}
	}

	var manifest packageManifestWithPackageCategories
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		return specerrors.ValidationErrors{
			specerrors.NewStructuredErrorf("file \"%s\" is invalid: %w", fsys.Path(manifestPath), errFailedToParseManifest)}
	}

	if manifest.Type != packageTypeIntegration {
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
			if _, isParent := parentCategories[dsCat]; isParent && !slices.Contains(manifest.Categories, dsCat) {
				missingCats = append(missingCats, dsCat)
			}
		}

		if len(missingCats) > 0 {
			sort.Strings(missingCats)
			dsManifestPath := path.Join("data_stream", dsName, "manifest.yml")
			errs = append(errs, specerrors.NewStructuredErrorf(
				"file \"%s\" is invalid: package manifest categories %v are missing parent categories %v from data stream \"%s\" (defined in \"%s\")",
				fsys.Path(manifestPath),
				manifest.Categories,
				missingCats,
				dsName,
				fsys.Path(dsManifestPath),
			))
		}
	}

	return errs
}
