// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package semantic

import (
	"fmt"
	"io/fs"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/elastic/package-spec/v3/code/go/internal/fspath"
	"github.com/elastic/package-spec/v3/code/go/pkg/specerrors"
)

var requiredMessageValues = []string{
	"_ingest.on_failure_processor_type",
	"_ingest.on_failure_processor_tag",
	"_ingest.on_failure_message",
	"_ingest.pipeline",
}

// ValidatePipelineOnFailure validates ingest pipeline global on_failure handlers.
func ValidatePipelineOnFailure(fsys fspath.FS) specerrors.ValidationErrors {

	var errs specerrors.ValidationErrors
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

		if vErrs := validatePipelineOnFailure(&pipeline, pipelineFile.fullFilePath); len(vErrs) > 0 {
			errs = append(errs, vErrs...)
		}
	}

	return errs
}

func validatePipelineOnFailure(pipeline *ingestPipeline, filename string) specerrors.ValidationErrors {
	var errs specerrors.ValidationErrors

	if e := checkSetEventKind(pipeline, filename); len(e) > 0 {
		errs = append(errs, e...)
	}
	if e := checkSetErrorMessage(pipeline, filename); len(e) > 0 {
		errs = append(errs, e...)
	}

	return errs
}

func checkSetEventKind(pipeline *ingestPipeline, filename string) specerrors.ValidationErrors {
	var errs specerrors.ValidationErrors
	var found bool

	for _, proc := range pipeline.OnFailure {
		if proc.Type != "set" {
			continue
		}
		if s, ok := proc.GetAttributeString("field"); !ok || s != "event.kind" {
			continue
		}

		found = true

		if s, ok := proc.GetAttributeString("value"); !ok || s != "pipeline_error" {
			errs = append(errs, specerrors.NewStructuredError(
				fmt.Errorf("file %q is invalid: pipeline on_failure handler must set event.kind to \"pipeline_error\"", filename),
				specerrors.CodePipelineOnFailureEventKind),
			)
		}

		break
	}

	if !found {
		errs = append(errs, specerrors.NewStructuredError(
			fmt.Errorf("file %q is invalid: pipeline on_failure handler must set event.kind to \"pipeline_error\"", filename),
			specerrors.CodePipelineOnFailureEventKind),
		)
	}

	return errs
}

func checkSetErrorMessage(pipeline *ingestPipeline, filename string) specerrors.ValidationErrors {
	var errs specerrors.ValidationErrors
	var found bool

	for _, proc := range pipeline.OnFailure {
		if proc.Type != "set" && proc.Type != "append" {
			continue
		}
		if s, ok := proc.GetAttributeString("field"); !ok || s != "error.message" {
			continue
		}

		found = true

		value, _ := proc.GetAttributeString("value")
		for _, reqMessageValue := range requiredMessageValues {
			if !strings.Contains(value, reqMessageValue) {
				errs = append(errs, specerrors.NewStructuredError(
					fmt.Errorf("file %q is invalid: pipeline on_failure error.message must include %q", filename, reqMessageValue),
					specerrors.CodePipelineOnFailureMessage),
				)
			}
		}

		break
	}

	if !found {
		errs = append(errs, specerrors.NewStructuredError(
			fmt.Errorf("file %q is invalid: pipeline on_failure handler must set error.message", filename),
			specerrors.CodePipelineOnFailureMessage),
		)
	}

	return errs
}
