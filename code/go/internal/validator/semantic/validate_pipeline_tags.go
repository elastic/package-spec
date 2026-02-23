// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package semantic

import (
	"fmt"
	"io/fs"

	"gopkg.in/yaml.v3"

	"github.com/elastic/package-spec/v3/code/go/internal/fspath"
	"github.com/elastic/package-spec/v3/code/go/pkg/specerrors"
)

// ValidatePipelineTags validates ingest pipeline processor tags.
func ValidatePipelineTags(fsys fspath.FS) specerrors.ValidationErrors {
	var errors specerrors.ValidationErrors
	pipelineFiles, err := listPipelineFiles(fsys)
	if err != nil {
		return specerrors.ValidationErrors{specerrors.NewStructuredError(err, specerrors.UnassignedCode)}
	}

	for _, pipelineFile := range pipelineFiles {
		content, err := fs.ReadFile(fsys, pipelineFile.filePath)
		if err != nil {
			return specerrors.ValidationErrors{specerrors.NewStructuredError(err, specerrors.UnassignedCode)}
		}

		var pipeline ingestPipeline
		if err = yaml.Unmarshal(content, &pipeline); err != nil {
			return specerrors.ValidationErrors{specerrors.NewStructuredError(err, specerrors.UnassignedCode)}
		}

		if vErrs := validatePipelineTags(&pipeline, pipelineFile.fullFilePath); len(vErrs) > 0 {
			errors = append(errors, vErrs...)
		}
	}

	return errors
}

func validatePipelineTags(pipeline *ingestPipeline, filename string) specerrors.ValidationErrors {
	var errors specerrors.ValidationErrors

	seen := map[string]struct{}{}
	for _, proc := range pipeline.Processors {
		procErrors := checkPipelineTag(&proc, seen, filename)
		errors = append(errors, procErrors...)
	}

	return errors
}

func checkPipelineTag(proc *processor, seen map[string]struct{}, filename string) specerrors.ValidationErrors {
	var errors specerrors.ValidationErrors

	for _, subProc := range proc.OnFailure {
		subErrors := checkPipelineTag(&subProc, seen, filename)
		errors = append(errors, subErrors...)
	}

	raw, ok := proc.Attributes["tag"]
	if !ok {
		errors = append(errors, specerrors.NewStructuredError(fmt.Errorf("file %q is invalid: %s processor at line %d missing required tag", filename, proc.Type, proc.position.line), specerrors.CodePipelineTagRequired))
		return errors
	}

	tag, ok := raw.(string)
	if !ok {
		errors = append(errors, specerrors.NewStructuredError(fmt.Errorf("file %q is invalid: %s processor at line %d has invalid tag value", filename, proc.Type, proc.position.line), specerrors.CodePipelineTagRequired))
		return errors
	}
	if tag == "" {
		errors = append(errors, specerrors.NewStructuredError(fmt.Errorf("file %q is invalid: %s processor at line %d has empty tag value", filename, proc.Type, proc.position.line), specerrors.CodePipelineTagRequired))
		return errors
	}

	if _, dup := seen[tag]; dup {
		errors = append(errors, specerrors.NewStructuredErrorf("file %q is invalid: %s processor at line %d has duplicate tag value: %q", filename, proc.Type, proc.position.line, tag))
		return errors
	}

	seen[tag] = struct{}{}

	return errors
}
