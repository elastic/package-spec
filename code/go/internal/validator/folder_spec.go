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

	"github.com/pkg/errors"

	ve "github.com/elastic/package-spec/code/go/internal/errors"
	"github.com/elastic/package-spec/code/go/internal/spectypes"
)

type Validator struct {
	spec       spectypes.ItemSpec
	pkg        *Package
	folderPath string

	totalSize     spectypes.FileSize
	totalContents int
}

func NewValidator(spec spectypes.ItemSpec, pkg *Package) *Validator {
	return newValidatorForPath(spec, pkg, ".")
}

func newValidatorForPath(spec spectypes.ItemSpec, pkg *Package, folderPath string) *Validator {
	return &Validator{
		spec:       spec,
		pkg:        pkg,
		folderPath: folderPath,
	}
}

func (v *Validator) Validate() ve.ValidationErrors {
	return v.validateSpec(v.spec, v.pkg, v.folderPath)
}

func (v *Validator) validateSpec(spec spectypes.ItemSpec, pkg *Package, folderPath string) ve.ValidationErrors {
	var errs ve.ValidationErrors
	files, err := fs.ReadDir(pkg, folderPath)
	if err != nil {
		errs = append(errs, errors.Wrapf(err, "could not read folder [%s]", pkg.Path(folderPath)))
		return errs
	}

	// This is not taking into account if the folder is for development. Enforce
	// this limit in all cases to avoid having to read too many files.
	if contentsLimit := spec.MaxTotalContents(); contentsLimit > 0 && len(files) > contentsLimit {
		errs = append(errs, errors.Errorf("folder [%s] exceeds the limit of %d files", pkg.Path(folderPath), contentsLimit))
		return errs
	}

	// Don't enable beta features for packages marked as GA.
	switch spec.Release() {
	case "", "ga": // do nothing
	case "beta":
		if pkg.Version.Major() > 0 && pkg.Version.Prerelease() == "" {
			errs = append(errs, errors.Errorf("spec for [%s] defines beta features which can't be enabled for packages with a stable semantic version", pkg.Path(folderPath)))
		} else {
			log.Printf("Warning: package with non-stable semantic version and active beta features (enabled in [%s]) can't be released as stable version.", pkg.Path(folderPath))
		}
	default:
		errs = append(errs, errors.Errorf("unsupport release level, supported values: beta, ga"))
	}

	for _, file := range files {
		fileName := file.Name()
		itemSpec, err := findItemSpec(spec, pkg.Name, fileName)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		if itemSpec == nil && spec.AdditionalContents() {
			// No spec found for current folder item, but we do allow additional contents in folder.
			if file.IsDir() {
				if !spec.DevelopmentFolder() && strings.Contains(fileName, "-") {
					errs = append(errs,
						fmt.Errorf(`file "%s" is invalid: directory name inside package %s contains -: %s`,
							pkg.Path(folderPath, fileName), pkg.Name, fileName))
				}
			}
			continue
		}

		if itemSpec == nil && !spec.AdditionalContents() {
			// No spec found for current folder item and we do not allow additional contents in folder.
			errs = append(errs, fmt.Errorf("item [%s] is not allowed in folder [%s]", fileName, pkg.Path(folderPath)))
			continue
		}

		if file.IsDir() {
			if !itemSpec.IsDir() {
				errs = append(errs, fmt.Errorf("[%s] is a folder but is expected to be a file", fileName))
				continue
			}

			subFolderPath := path.Join(folderPath, fileName)
			itemValidator := newValidatorForPath(itemSpec, pkg, subFolderPath)
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
				errs = append(errs, fmt.Errorf("[%s] is a file but is expected to be a folder", pkg.Path(fileName)))
				continue
			}

			itemPath := path.Join(folderPath, file.Name())
			itemValidationErrs := validateFile(itemSpec, pkg, itemPath)
			for _, ive := range itemValidationErrs {
				errs = append(errs, errors.Wrapf(ive, "file \"%s\" is invalid", pkg.Path(itemPath)))
			}

			info, err := fs.Stat(pkg, itemPath)
			if err != nil {
				errs = append(errs, errors.Wrapf(err, "failed to obtain file size for \"%s\"", pkg.Path(itemPath)))
			} else {
				v.totalContents++
				v.totalSize += spectypes.FileSize(info.Size())
			}
		}
	}

	if sizeLimit := spec.MaxTotalSize(); sizeLimit > 0 && v.totalSize > sizeLimit {
		errs = append(errs, errors.Errorf("folder [%s] exceeds the total size limit of %s", pkg.Path(folderPath), sizeLimit))
	}

	// validate that required items in spec are all accounted for
	for _, itemSpec := range spec.Contents() {
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
				err = fmt.Errorf("expecting to find [%s] %s in folder [%s]", itemSpec.Name(), itemSpec.Type(), pkg.Path(folderPath))
			} else if itemSpec.Pattern() != "" {
				err = fmt.Errorf("expecting to find %s matching pattern [%s] in folder [%s]", itemSpec.Type(), itemSpec.Pattern(), pkg.Path(folderPath))
			}
			errs = append(errs, err)
		}
	}
	return errs
}

func findItemSpec(s spectypes.ItemSpec, packageName string, folderItemName string) (spectypes.ItemSpec, error) {
	for _, itemSpec := range s.Contents() {
		if itemSpec.Name() != "" && itemSpec.Name() == folderItemName {
			return itemSpec, nil
		}
		if itemSpec.Pattern() != "" {
			isMatch, err := regexp.MatchString(strings.ReplaceAll(itemSpec.Pattern(), "{PACKAGE_NAME}", packageName), folderItemName)
			if err != nil {
				return nil, errors.Wrap(err, "invalid folder item spec pattern")
			}
			if isMatch {
				var isForbidden bool
				for _, forbidden := range itemSpec.ForbiddenPatterns() {
					isForbidden, err = regexp.MatchString(forbidden, folderItemName)
					if err != nil {
						return nil, errors.Wrap(err, "invalid forbidden pattern for folder item")
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
