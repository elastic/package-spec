// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package semantic

import (
	"fmt"
	"io/fs"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/elastic/package-spec/v3/code/go/internal/fspath"
	"github.com/elastic/package-spec/v3/code/go/pkg/specerrors"
)

// grokToken is what Elasticsearch's grok reads as a `%{...}` token, transcribed from
// GROK_PATTERN in org.elasticsearch.grok.Grok and anchored at the start of the input.
// The pattern name is `[A-z0-9]+` there too: the range runs from `A` to `z` and so also
// admits `[`, `\`, `]`, `^`, `_` and the backtick.
//
// Anything else that starts with `%{` is not a token. Grok does not report that, it leaves
// the text in the regular expression as-is, so the processor compiles and runs and simply
// never captures what the author wrote.
var grokToken = regexp.MustCompile(`^%\{[A-z0-9]+(?::[[:alnum:]@\[\]_:.-]+)?(?:=(?:[^{}]+|\.+)+)?\}`)

// ValidatePipelineGrok validates that every `%{` in a grok processor is a grok token.
func ValidatePipelineGrok(fsys fspath.FS) specerrors.ValidationErrors {
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

		if vErrs := validatePipelineGrok(&pipeline, pipelineFile.fullFilePath); len(vErrs) > 0 {
			errors = append(errors, vErrs...)
		}
	}

	return errors
}

func validatePipelineGrok(pipeline *ingestPipeline, filename string) specerrors.ValidationErrors {
	var errors specerrors.ValidationErrors

	for _, proc := range pipeline.Processors {
		errors = append(errors, checkPipelineGrok(&proc, filename)...)
	}
	for _, proc := range pipeline.OnFailure {
		errors = append(errors, checkPipelineGrok(&proc, filename)...)
	}

	return errors
}

func checkPipelineGrok(proc *processor, filename string) specerrors.ValidationErrors {
	var errors specerrors.ValidationErrors

	for _, subProc := range proc.OnFailure {
		errors = append(errors, checkPipelineGrok(&subProc, filename)...)
	}

	// A foreach processor wraps another processor, which may be a grok. It is decoded
	// as part of the foreach's attributes, so it has to be lifted out here.
	if proc.Type == "foreach" {
		if inner, ok := nestedProcessor(proc.Attributes["processor"], proc.position); ok {
			errors = append(errors, checkPipelineGrok(&inner, filename)...)
		}
	}

	if proc.Type != "grok" {
		return errors
	}

	if patterns, ok := proc.Attributes["patterns"].([]any); ok {
		for i, pattern := range patterns {
			if text, ok := pattern.(string); ok {
				where := fmt.Sprintf("patterns[%d]", i)
				errors = append(errors, checkGrokText(text, where, proc, filename)...)
			}
		}
	}

	if definitions, ok := proc.Attributes["pattern_definitions"].(map[string]any); ok {
		names := make([]string, 0, len(definitions))
		for name := range definitions {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			if text, ok := definitions[name].(string); ok {
				where := fmt.Sprintf("pattern_definitions[%q]", name)
				errors = append(errors, checkGrokText(text, where, proc, filename)...)
			}
		}
	}

	return errors
}

// checkGrokText reports every `%{` in text that grok would not read as a token.
func checkGrokText(text, where string, proc *processor, filename string) specerrors.ValidationErrors {
	var errors specerrors.ValidationErrors

	for offset := 0; ; {
		start := strings.Index(text[offset:], "%{")
		if start < 0 {
			break
		}
		start += offset
		if match := grokToken.FindStringIndex(text[start:]); match != nil {
			offset = start + match[1]
			continue
		}
		errors = append(errors, specerrors.NewStructuredError(
			fmt.Errorf("file %q is invalid: grok processor at line %d has %q in %s, which is not a grok token and is matched as literal text",
				filename, proc.position.line, malformedTokenExcerpt(text[start:]), where),
			specerrors.CodePipelineGrokMalformedToken))
		offset = start + len("%{")
	}

	return errors
}

// malformedTokenExcerpt is the text from a `%{` that failed to parse as a token up to and
// including the first `}` after it, or a bounded prefix when there is none.
func malformedTokenExcerpt(text string) string {
	const limit = 60
	if end := strings.IndexByte(text, '}'); end >= 0 && end < limit {
		return text[:end+1]
	}
	if len(text) > limit {
		return text[:limit] + "..."
	}
	return text
}

// nestedProcessor rebuilds a processor from the single-key map a foreach carries under
// `processor`. Its position is lost in decoding, so the enclosing processor's is used.
func nestedProcessor(value any, position position) (processor, bool) {
	wrapper, ok := value.(map[string]any)
	if !ok || len(wrapper) != 1 {
		return processor{}, false
	}
	for procType, raw := range wrapper {
		attributes, ok := raw.(map[string]any)
		if !ok {
			return processor{}, false
		}
		inner := processor{Type: procType, Attributes: attributes, position: position}
		if onFailure, ok := attributes["on_failure"].([]any); ok {
			for _, entry := range onFailure {
				if sub, ok := nestedProcessor(entry, position); ok {
					inner.OnFailure = append(inner.OnFailure, sub)
				}
			}
		}
		return inner, true
	}
	return processor{}, false
}
