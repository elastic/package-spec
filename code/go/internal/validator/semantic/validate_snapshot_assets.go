// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package semantic

import (
	"fmt"
	"log"
	"path"

	"github.com/Masterminds/semver/v3"
	"github.com/pkg/errors"

	ve "github.com/elastic/package-spec/code/go/internal/errors"
	"github.com/elastic/package-spec/code/go/internal/fspath"
	"github.com/elastic/package-spec/code/go/internal/pkgpath"
	"github.com/elastic/package-spec/code/go/internal/validator/common"
)

const elasticPrereleaseTag = "SNAPSHOT"

var assetsToCheck = []string{
	"dashboard",
	"visualization",
	"lens",
	"map",
}

// ValidateSnapshotVersionsInAssets validates additional restrictions on the Elastic stack versions used to generate assets.
func ValidateSnapshotVersionsInAssets(fsys fspath.FS) ve.ValidationErrors {
	warningsAsErrors := common.IsDefinedWarningsAsErrors()
	var errs ve.ValidationErrors

	manifestVersion, err := readManifestVersion(fsys)
	if err != nil {
		return ve.ValidationErrors{err}
	}

	allow_snapshot, err := readAllowSnapshotManifest(fsys)
	if err != nil {
		return ve.ValidationErrors{err}
	}

	// versions from dashboards, visualizations, lens, etc.
	// prereleased versions allowed to contain -SNAPSHOT (no restrictions?)
	packageVersion, err := semver.NewVersion(manifestVersion)
	if packageVersion.Major() == 0 || packageVersion.Prerelease() != "" {
		// no retrictions, it can contain -SNAPSHOT
		return nil
	}

	// stable versions allowed to contain -SNAPSHOT if allow_snapshot is defined
	if allow_snapshot {
		return nil
	}

	// stable versions not allowed to contain -SNAPSHOT
	for _, asset := range assetsToCheck {
		filePaths := path.Join("kibana", asset, "*.json")
		objectFiles, err := pkgpath.Files(fsys, filePaths)
		if err != nil {
			// errs = append(errs, errors.Wrapf(err, "error finding %s files", asset))
			continue
		}

		for _, objectFile := range objectFiles {
			filePath := objectFile.Path()

			snapshot, version, err := usingSnapshotVersion(objectFile, asset)
			if err != nil {
				errs = append(errs, err)
				continue
			}

			if snapshot {
				message := fmt.Sprintf("Warning: prerelease version found in %s %s: %s", asset, filePath, version)
				if warningsAsErrors {
					errs = append(errs, errors.New(message))
				} else {
					log.Printf(message)
				}
			}
		}
	}

	return nil
}

func usingSnapshotVersion(objectFile pkgpath.File, asset string) (bool, string, error) {
	versionReference, err := objectFile.Values(`$.migrationVersion.` + asset)
	if err != nil {
		// some assets do not have migrationVersion field
		// return false, errors.New("no migrationVersion field found")
		return false, "", nil
	}
	version, ok := versionReference.(string)
	if !ok {
		return false, "", errors.Errorf("conversion error to string %s", versionReference)
	}

	semVersion, err := semver.NewVersion(version)
	if err != nil {
		return false, "", err
	}
	return semVersion.Prerelease() == elasticPrereleaseTag, version, nil
}

func readAllowSnapshotManifest(fsys fspath.FS) (bool, error) {
	manifestPath := "manifest.yml"
	f, err := pkgpath.Files(fsys, manifestPath)
	if err != nil {
		return false, errors.Wrap(err, "can't locate manifest file")
	}

	if len(f) != 1 {
		return false, errors.New("single manifest file expected")
	}

	val, err := f[0].Values("$.allow_snapshot")
	if err != nil {
		return false, nil
	}

	bVal, ok := val.(bool)
	if !ok {
		return false, errors.New("allow_snapshot unexpected value")
	}
	return bVal, nil
}
