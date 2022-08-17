// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package validator

import (
	"fmt"
	"io/fs"
	"io/ioutil"
	"log"
	"path"
	"regexp"
	"strings"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"

	ve "github.com/elastic/package-spec/code/go/internal/errors"
	"github.com/elastic/package-spec/code/go/internal/spectypes"
	"github.com/elastic/package-spec/code/go/internal/validator/common"
)

const (
	itemTypeFile   = "file"
	itemTypeFolder = "folder"

	visibilityTypePublic  = "public"
	visibilityTypePrivate = "private"
)

type folderSpec struct {
	fs       fs.FS
	specPath string
	commonSpec

	// These "validation-time" fields don't actually belong to the spec, storing
	// them here for convenience by now.
	totalSize     spectypes.FileSize
	totalContents int
}

func (s *folderSpec) load(fs fs.FS, specPath string) error {
	specFile, err := fs.Open(specPath)
	if err != nil {
		return errors.Wrap(err, "could not open folder specification file")
	}
	defer specFile.Close()

	data, err := ioutil.ReadAll(specFile)
	if err != nil {
		return errors.Wrap(err, "could not read folder specification file")
	}

	var wrapper struct {
		Spec *commonSpec `yaml:"spec"`
	}
	wrapper.Spec = &s.commonSpec
	if err := yaml.Unmarshal(data, &wrapper); err != nil {
		return errors.Wrap(err, "could not parse folder specification file")
	}

	err = setDefaultValues(&s.commonSpec)
	if err != nil {
		return errors.Wrap(err, "could not set default values")
	}

	propagateContentLimits(&s.commonSpec)

	s.fs = fs
	s.specPath = specPath
	return nil
}

func (s *folderSpec) validate(pkg *Package, folderPath string) ve.ValidationErrors {
	var errs ve.ValidationErrors
	files, err := fs.ReadDir(pkg, folderPath)
	if err != nil {
		errs = append(errs, errors.Wrapf(err, "could not read folder [%s]", pkg.Path(folderPath)))
		return errs
	}

	// This is not taking into account if the folder is for development. Enforce
	// this limit in all cases to avoid having to read too many files.
	if contentsLimit := s.Limits.TotalContentsLimit; contentsLimit > 0 && len(files) > contentsLimit {
		errs = append(errs, errors.Errorf("folder [%s] exceeds the limit of %d files", pkg.Path(folderPath), contentsLimit))
		return errs
	}

	// Don't enable beta features for packages marked as GA.
	switch s.Release {
	case "", "ga": // do nothing
	case "beta":
		if pkg.Version.Major() > 0 && pkg.Version.Prerelease() == "" {
			errs = append(errs, errors.Errorf("spec for [%s] defines beta features which can't be enabled for packages with a stable semantic version", pkg.Path(folderPath)))
		} else {
			if common.IsDefinedWarningsAsErrors() {
				errs = append(errs, errors.Errorf("Warning: package with non-stable semantic version and active beta features (enabled in [%s]) can't be released as stable version.", pkg.Path(folderPath)))
			} else {
				log.Printf("Warning: package with non-stable semantic version and active beta features (enabled in [%s]) can't be released as stable version.", pkg.Path(folderPath))
			}
		}
	default:
		errs = append(errs, errors.Errorf("unsupport release level, supported values: beta, ga"))
	}

	for _, file := range files {
		fileName := file.Name()
		itemSpec, err := s.findItemSpec(pkg.Name, fileName)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		if itemSpec == nil && s.AdditionalContents {
			// No spec found for current folder item, but we do allow additional contents in folder.
			if file.IsDir() {
				if !s.DevelopmentFolder && strings.Contains(fileName, "-") {
					errs = append(errs,
						fmt.Errorf(`file "%s" is invalid: directory name inside package %s contains -: %s`,
							pkg.Path(folderPath, fileName), pkg.Name, fileName))
				}
			}
			continue
		}

		if itemSpec == nil && !s.AdditionalContents {
			// No spec found for current folder item and we do not allow additional contents in folder.
			errs = append(errs, fmt.Errorf("item [%s] is not allowed in folder [%s]", fileName, pkg.Path(folderPath)))
			continue
		}

		if itemSpec != nil && itemSpec.Visibility != visibilityTypePrivate && itemSpec.Visibility != visibilityTypePublic {
			errs = append(errs, fmt.Errorf("item [%s] visibility is expected to be private or public, not [%s]", fileName, itemSpec.Visibility))
			continue
		}

		if file.IsDir() {
			if !itemSpec.isSameType(file) {
				errs = append(errs, fmt.Errorf("[%s] is a folder but is expected to be a file", fileName))
				continue
			}

			if itemSpec.Ref == "" && itemSpec.Contents == nil {
				// No recursive validation needed
				continue
			}

			var subFolderSpec folderSpec
			// Inherit limits from parent directory.
			subFolderSpec.Limits = s.Limits
			if itemSpec.Ref != "" {
				subFolderSpecPath := path.Join(path.Dir(s.specPath), itemSpec.Ref)
				err := subFolderSpec.load(s.fs, subFolderSpecPath)
				if err != nil {
					errs = append(errs, err)
					continue
				}
			} else if itemSpec.Contents != nil {
				subFolderSpec.fs = s.fs
				subFolderSpec.specPath = s.specPath
				subFolderSpec.commonSpec.AdditionalContents = itemSpec.AdditionalContents
				subFolderSpec.commonSpec.Contents = itemSpec.Contents
			}

			// Subfolders of development folders are also considered development folders.
			if s.DevelopmentFolder {
				subFolderSpec.DevelopmentFolder = true
			}

			subFolderPath := path.Join(folderPath, fileName)
			subErrs := subFolderSpec.validate(pkg, subFolderPath)
			if len(subErrs) > 0 {
				errs = append(errs, subErrs...)
			}

			// Don't count files in development folders.
			if !subFolderSpec.DevelopmentFolder {
				s.totalContents += subFolderSpec.totalContents
				s.totalSize += subFolderSpec.totalSize
			}
		} else {
			if !itemSpec.isSameType(file) {
				errs = append(errs, fmt.Errorf("[%s] is a file but is expected to be a folder", pkg.Path(fileName)))
				continue
			}

			itemPath := path.Join(folderPath, file.Name())
			itemValidationErrs := itemSpec.validate(s.fs, pkg, s.specPath, itemPath)
			if itemValidationErrs != nil {
				for _, ive := range itemValidationErrs {
					errs = append(errs, errors.Wrapf(ive, "file \"%s\" is invalid", pkg.Path(itemPath)))
				}
			}

			info, err := fs.Stat(pkg, itemPath)
			if err != nil {
				errs = append(errs, errors.Wrapf(err, "failed to obtain file size for \"%s\"", pkg.Path(itemPath)))
			} else {
				s.totalContents++
				s.totalSize += spectypes.FileSize(info.Size())
			}
		}
	}

	if sizeLimit := s.Limits.TotalSizeLimit; sizeLimit > 0 && s.totalSize > sizeLimit {
		errs = append(errs, errors.Errorf("folder [%s] exceeds the total size limit of %s", pkg.Path(folderPath), sizeLimit))
	}

	// validate that required items in spec are all accounted for
	for _, itemSpec := range s.Contents {
		if !itemSpec.Required {
			continue
		}

		fileFound, err := itemSpec.matchingFileExists(files)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		if !fileFound {
			var err error
			if itemSpec.Name != "" {
				err = fmt.Errorf("expecting to find [%s] %s in folder [%s]", itemSpec.Name, itemSpec.ItemType, pkg.Path(folderPath))
			} else if itemSpec.Pattern != "" {
				err = fmt.Errorf("expecting to find %s matching pattern [%s] in folder [%s]", itemSpec.ItemType, itemSpec.Pattern, pkg.Path(folderPath))
			}
			errs = append(errs, err)
		}
	}
	return errs
}

func (s *folderSpec) findItemSpec(packageName string, folderItemName string) (*folderItemSpec, error) {
	for _, itemSpec := range s.Contents {
		if itemSpec.Name != "" && itemSpec.Name == folderItemName {
			return &itemSpec, nil
		}
		if itemSpec.Pattern != "" {
			isMatch, err := regexp.MatchString(strings.ReplaceAll(itemSpec.Pattern, "{PACKAGE_NAME}", packageName), folderItemName)
			if err != nil {
				return nil, errors.Wrap(err, "invalid folder item spec pattern")
			}
			if isMatch {
				var isForbidden bool
				for _, forbidden := range itemSpec.ForbiddenPatterns {
					isForbidden, err = regexp.MatchString(forbidden, folderItemName)
					if err != nil {
						return nil, errors.Wrap(err, "invalid forbidden pattern for folder item")
					}

					if isForbidden {
						break
					}
				}

				if !isForbidden {
					return &itemSpec, nil
				}
			}
		}
	}

	// No item spec found
	return nil, nil
}
