// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package validator

import (
	"fmt"
	"io/fs"
	"log"
	"path"
	"regexp"
	"strings"

	"github.com/elastic/package-spec/v2/code/go/internal/packages"
	"github.com/elastic/package-spec/v2/code/go/internal/spectypes"
	"github.com/elastic/package-spec/v2/code/go/internal/validator/common"
	ve "github.com/elastic/package-spec/v2/code/go/pkg/errors"
)

type validator struct {
	spec       spectypes.ItemSpec
	pkg        *packages.Package
	folderPath string

	totalSize     spectypes.FileSize
	totalContents int
}

func newValidator(spec spectypes.ItemSpec, pkg *packages.Package) *validator {
	return newValidatorForPath(spec, pkg, ".")
}

func newValidatorForPath(spec spectypes.ItemSpec, pkg *packages.Package, folderPath string) *validator {
	return &validator{
		spec:       spec,
		pkg:        pkg,
		folderPath: folderPath,
	}
}

func (v *validator) Validate() ve.ValidationErrors {
	var errs ve.ValidationErrors
	files, err := fs.ReadDir(v.pkg, v.folderPath)
	if err != nil {
		errs = append(errs, fmt.Errorf("could not read folder [%s]: %w", v.pkg.Path(v.folderPath), err))
		return errs
	}

	// This is not taking into account if the folder is for development. Enforce
	// this limit in all cases to avoid having to read too many files.
	if contentsLimit := v.spec.MaxTotalContents(); contentsLimit > 0 && len(files) > contentsLimit {
		errs = append(errs, fmt.Errorf("folder [%s] exceeds the limit of %d files", v.pkg.Path(v.folderPath), contentsLimit))
		return errs
	}

	// Don't enable beta features for packages marked as GA.
	switch v.spec.Release() {
	case "", "ga": // do nothing
	case "beta":
		if v.pkg.Version.Major() > 0 && v.pkg.Version.Prerelease() == "" {
			errs = append(errs, fmt.Errorf("spec for [%s] defines beta features which can't be enabled for packages with a stable semantic version", v.pkg.Path(v.folderPath)))
		} else {
			message := fmt.Sprintf("Warning: package with non-stable semantic version and active beta features (enabled in [%s]) can't be released as stable version.", v.pkg.Path(v.folderPath))
			if common.IsDefinedWarningsAsErrors() {
				errs = append(errs, fmt.Errorf(message))
			} else {
				log.Printf(message)
			}
		}
	default:
		errs = append(errs, fmt.Errorf("unsupport release level, supported values: beta, ga"))
	}

	for _, file := range files {
		fileName := file.Name()
		itemSpec, err := v.findItemSpec(fileName)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		if itemSpec == nil && v.spec.AdditionalContents() {
			// No spec found for current folder item, but we do allow additional contents in folder.
			if file.IsDir() {
				if !v.spec.DevelopmentFolder() && strings.Contains(fileName, "-") {
					errs = append(errs,
						fmt.Errorf(`file "%s" is invalid: directory name inside package %s contains -: %s`,
							v.pkg.Path(v.folderPath, fileName), v.pkg.Name, fileName))
				}
			}
			continue
		}

		if itemSpec == nil && !v.spec.AdditionalContents() {
			// No spec found for current folder item and we do not allow additional contents in folder.
			errs = append(errs, fmt.Errorf("item [%s] is not allowed in folder [%s]", fileName, v.pkg.Path(v.folderPath)))
			continue
		}

		if file.IsDir() {
			if !itemSpec.IsDir() {
				errs = append(errs, fmt.Errorf("[%s] is a folder but is expected to be a file", fileName))
				continue
			}

			subFolderPath := path.Join(v.folderPath, fileName)
			itemValidator := newValidatorForPath(itemSpec, v.pkg, subFolderPath)
			subErrs := itemValidator.Validate()
			if len(subErrs) > 0 {
				errs = append(errs, subErrs...)
			}

			// Don't count files in development folders.
			if !itemSpec.DevelopmentFolder() {
				v.totalContents += itemValidator.totalContents
				v.totalSize += itemValidator.totalSize
			}
		} else {
			if itemSpec.IsDir() {
				errs = append(errs, fmt.Errorf("[%s] is a file but is expected to be a folder", v.pkg.Path(fileName)))
				continue
			}

			itemPath := path.Join(v.folderPath, file.Name())
			itemValidationErrs := validateFile(itemSpec, v.pkg, itemPath)
			for _, ive := range itemValidationErrs {
				errs = append(errs, fmt.Errorf("file \"%s\" is invalid: %w", v.pkg.Path(itemPath), ive))
			}

			info, err := fs.Stat(v.pkg, itemPath)
			if err != nil {
				errs = append(errs, fmt.Errorf("failed to obtain file size for \"%s\": %w", v.pkg.Path(itemPath), err))
			} else {
				v.totalContents++
				v.totalSize += spectypes.FileSize(info.Size())
			}
		}
	}

	if sizeLimit := v.spec.MaxTotalSize(); sizeLimit > 0 && v.totalSize > sizeLimit {
		errs = append(errs, fmt.Errorf("folder [%s] exceeds the total size limit of %s", v.pkg.Path(v.folderPath), sizeLimit))
	}

	// validate that required items in spec are all accounted for
	for _, itemSpec := range v.spec.Contents() {
		if !itemSpec.Required() {
			continue
		}

		fileFound, err := matchingFileExists(itemSpec, files)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		if !fileFound {
			var err error
			if itemSpec.Name() != "" {
				err = fmt.Errorf("expecting to find [%s] %s in folder [%s]", itemSpec.Name(), itemSpec.Type(), v.pkg.Path(v.folderPath))
			} else if itemSpec.Pattern() != "" {
				err = fmt.Errorf("expecting to find %s matching pattern [%s] in folder [%s]", itemSpec.Type(), itemSpec.Pattern(), v.pkg.Path(v.folderPath))
			}
			errs = append(errs, err)
		}
	}
	return errs
}

func (v *validator) findItemSpec(folderItemName string) (spectypes.ItemSpec, error) {
	for _, itemSpec := range v.spec.Contents() {
		if itemSpec.Name() != "" && itemSpec.Name() == folderItemName {
			return itemSpec, nil
		}
		if itemSpec.Pattern() != "" {
			isMatch, err := regexp.MatchString(strings.ReplaceAll(itemSpec.Pattern(), "{PACKAGE_NAME}", v.pkg.Name), folderItemName)
			if err != nil {
				return nil, fmt.Errorf("invalid folder item spec pattern: %w", err)
			}
			if isMatch {
				var isForbidden bool
				for _, forbidden := range itemSpec.ForbiddenPatterns() {
					isForbidden, err = regexp.MatchString(forbidden, folderItemName)
					if err != nil {
						return nil, fmt.Errorf("invalid forbidden pattern for folder item: %w", err)
					}

					if isForbidden {
						break
					}
				}

				if !isForbidden {
					return itemSpec, nil
				}
			}
		}
	}

	// No item spec found
	return nil, nil
}
