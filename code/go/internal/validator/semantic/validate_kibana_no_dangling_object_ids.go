// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package semantic

import (
	"fmt"
	"path"
	"slices"

	ve "github.com/elastic/package-spec/v2/code/go/internal/errors"
	"github.com/elastic/package-spec/v2/code/go/internal/fspath"
	"github.com/elastic/package-spec/v2/code/go/internal/pkgpath"
	pve "github.com/elastic/package-spec/v2/code/go/pkg/errors"
)

type objectReference struct {
	objectType string
	objectID   string
	filePath   string
}

var exceptionAssets = []string{
	"index-pattern", // https://github.com/elastic/elastic-package/blob/9d80c6cc282d04f521cd763abd42d529f9679cce/internal/export/transform_filter.go#L14
}

// ValidateKibanaNoDanglingObjectIDs returns validation errors if there are any
// dangling references to Kibana objects in any Kibana object files. That is, it
// returns validation errors if a Kibana object file in the package references another
// Kibana object with ID i, but no Kibana object file for object ID i is found in the
// package.
func ValidateKibanaNoDanglingObjectIDs(fsys fspath.FS) pve.ValidationErrors {
	var errs pve.ValidationErrors

	installedIDs := []objectReference{}
	referencedIDs := []objectReference{}

	filePaths := path.Join("kibana", "*", "*.json")
	objectFiles, err := pkgpath.Files(fsys, filePaths)
	if err != nil {
		vError := ve.NewStructuredError(
			fmt.Errorf("error finding Kibana object files: %w", err),
			"kibana", // FIXME other file path name?
			"",
			ve.Critical,
		)
		errs = append(errs, vError)
		return errs
	}
	for _, objectFile := range objectFiles {
		filePath := objectFile.Path()

		currentReference, err := getCurrentObjectReference(objectFile, fsys.Path(filePath))
		if err != nil {
			vError := ve.NewStructuredError(
				fmt.Errorf("unable to create reference from file [%s]: %w", fsys.Path(filePath), err),
				filePath,
				"",
				ve.Critical,
			)
			errs = append(errs, vError)
		}

		installedIDs = append(installedIDs, currentReference)

		referencedObjects, err := getReferencesListFromCurrentObject(objectFile, fsys.Path(filePath))
		if err != nil {
			vError := ve.NewStructuredError(
				fmt.Errorf("unable to create referenced objects from file [%s]: %w", fsys.Path(filePath), err),
				filePath,
				"",
				ve.Critical,
			)
			errs = append(errs, vError)
			continue
		}

		referencedIDs = append(referencedIDs, referencedObjects...)
	}

	if len(referencedIDs) == 0 {
		return errs
	}

	for _, reference := range referencedIDs {
		// look for installed IDs
		found := slices.ContainsFunc(installedIDs, func(elem objectReference) bool {
			if reference.objectID != elem.objectID {
				return false
			}
			if reference.objectType != elem.objectType {
				return false
			}
			return true
		})
		if !found {
			vError := ve.NewDanglingObjectIDError(
				reference.objectID,
				reference.objectType,
				reference.filePath,
			)
			errs = append(errs, vError)
		}
	}

	return errs
}

func getCurrentObjectReference(asset pkgpath.File, filePath string) (objectReference, error) {
	var reference objectReference

	valueID, err := asset.Values("$.id")
	if err != nil {
		return reference, fmt.Errorf("unable to get ID field : %w", err)
	}
	stringValueID, ok := valueID.(string)
	if !ok {
		return reference, fmt.Errorf("expect value ID to be a string: %w", err)
	}

	valueType, err := asset.Values("$.type")
	if err != nil {
		return reference, fmt.Errorf("unable to get Type field : %w", err)
	}
	stringValueType, ok := valueType.(string)
	if !ok {
		return reference, fmt.Errorf("expect value Type to be a string: %w", err)
	}

	reference = objectReference{
		objectID:   stringValueID,
		objectType: stringValueType,
		filePath:   filePath,
	}

	return reference, nil
}

func getReferencesListFromCurrentObject(asset pkgpath.File, filePath string) ([]objectReference, error) {
	referencedIDs := []objectReference{}
	objectReferences, err := asset.Values(`$.references`)
	if err != nil {
		// no references key in dashboard json
		return referencedIDs, nil
	}

	references, err := filterReferences(objectReferences, exceptionAssets)
	if err != nil {
		return nil, fmt.Errorf("error getting references: %w", err)
	}

	if len(references) == 0 {
		return referencedIDs, nil
	}

	for _, reference := range references {
		referencedIDs = append(referencedIDs, objectReference{
			objectID:   reference.ID,
			objectType: reference.Type,
			filePath:   filePath,
		})
	}

	return referencedIDs, nil
}

func filterReferences(val interface{}, exceptions []string) ([]reference, error) {
	allReferences, err := toReferenceSlice(val)
	if err != nil {
		return []reference{}, fmt.Errorf("unable to convert references: %w", err)
	}

	if len(allReferences) == 0 {
		return []reference{}, nil
	}

	var references []reference
	for _, reference := range allReferences {
		if slices.Contains(exceptions, reference.Type) {
			continue
		}
		references = append(references, reference)

	}
	return references, nil

}
