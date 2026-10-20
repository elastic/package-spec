// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package semantic

import (
	"errors"
	"fmt"
	"io/fs"
	"path"

	"gopkg.in/yaml.v3"

	"github.com/elastic/package-spec/v3/code/go/internal/fspath"
	"github.com/elastic/package-spec/v3/code/go/pkg/specerrors"
)

const workflowFolder = "kibana/workflow"

type workflow struct {
	Enabled *bool `yaml:"enabled"`
}

// ValidateWorkflowsEnabled warns when a workflow explicitly sets enabled: true.
// Fleet always installs package workflows disabled.
//
// TODO: temporary rule while workflows are imported by hand; revisit/remove once
// elastic-package supports exporting workflows (elastic/elastic-package#3475).
func ValidateWorkflowsEnabled(fsys fspath.FS) specerrors.ValidationErrors {
	entries, err := fs.ReadDir(fsys, workflowFolder)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return specerrors.ValidationErrors{specerrors.NewStructuredErrorf("error reading %s: %v", workflowFolder, err)}
	}

	var errs specerrors.ValidationErrors

	for _, entry := range entries {
		if entry.IsDir() || path.Ext(entry.Name()) != ".yml" {
			continue
		}

		filePath := path.Join(workflowFolder, entry.Name())
		fullPath := fsys.Path(filePath)

		b, err := fs.ReadFile(fsys, filePath)
		if err != nil {
			errs = append(errs, specerrors.NewStructuredErrorf("error reading file %s: %v", fullPath, err))
			continue
		}

		var w workflow
		if err := yaml.Unmarshal(b, &w); err != nil {
			errs = append(errs, specerrors.NewStructuredErrorf("error unmarshaling file %s: %v", fullPath, err))
			continue
		}

		if w.Enabled != nil && *w.Enabled {
			errs = append(errs, specerrors.NewStructuredError(
				fmt.Errorf("file \"%s\" is invalid: workflow should set enabled: false; package workflows are installed disabled", fullPath),
				specerrors.CodeWorkflowEnabled,
			))
		}
	}

	return errs
}
