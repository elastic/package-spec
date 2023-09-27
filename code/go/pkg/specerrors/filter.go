// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package specerrors

import (
	"fmt"
	"io/fs"

	"gopkg.in/yaml.v3"
)

const configPath = "validation.yml"

// Filter represents the collection of processors to be applied over validation errors
type Filter struct {
	processors []Processor
}

// FilterResult represents the errors that have been processed and removed from the filter
type FilterResult struct {
	Processed error
	Removed   error
}

// Run runs all the processors over all the validation errors and return the filtered ones
func (r *Filter) Run(allErrors ValidationErrors) (FilterResult, error) {
	newErrors := allErrors
	var allFiltered ValidationErrors

	for _, p := range r.processors {
		result, err := p.Process(newErrors)
		if err != nil {
			return FilterResult{Processed: allErrors, Removed: nil}, err
		}
		newErrors = result.Processed
		allFiltered.Append(result.Removed)
	}

	return FilterResult{
		Processed: nilOrValidationErrors(newErrors),
		Removed:   nilOrValidationErrors(allFiltered),
	}, nil
}

func nilOrValidationErrors(errs ValidationErrors) error {
	if len(errs) == 0 {
		return nil
	}
	return errs
}

// AddProcessors allows to add custom processors to the runner
func (r *Filter) AddProcessors(items []Processor) {
	r.processors = append(r.processors, items...)
}

// ConfigFilter represents the linter configuration file
type ConfigFilter struct {
	Errors Processors `yaml:"errors"`
}

// Processors represents the list of processors in the configuration file
type Processors struct {
	ExcludeChecks []string `yaml:"exclude_checks"`
}

// LoadConfigFilter reads the config file and returns a ConfigFilter struct
func LoadConfigFilter(fsys fs.FS) (*ConfigFilter, error) {
	yamlFile, err := fs.ReadFile(fsys, configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", configPath, err)
	}
	var config ConfigFilter
	err = yaml.Unmarshal(yamlFile, &config)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config file %s: %w", configPath, err)
	}
	return &config, nil
}

// NewFilter creates a new filter given a configuration
func NewFilter(config *ConfigFilter) *Filter {
	var filters []Processor
	for _, code := range config.Errors.ExcludeChecks {
		exclude := NewExcludeCheck(code)
		filters = append(filters, *exclude)
	}

	runner := Filter{
		processors: filters,
	}

	return &runner
}
