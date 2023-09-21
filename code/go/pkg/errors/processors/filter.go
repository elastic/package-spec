// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package processors

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"

	"github.com/elastic/package-spec/v2/code/go/pkg/errors"
)

// Filter represents the collection of processors to be applied over validation errors
type Filter struct {
	processors []Processor
}

// Run runs all the processors over all the validation errors and return the filtered ones
func (r *Filter) Run(errors errors.ValidationErrors) (errors.ValidationErrors, error) {
	newErrors := errors
	var err error
	for _, p := range r.processors {
		newErrors, err = p.Process(newErrors)
		if err != nil {
			return errors, err
		}
	}
	return newErrors, nil
}

// AddProcessors allows to add custom processors to the runner
func (r *Filter) AddProcessors(items []Processor) {
	r.processors = append(r.processors, items...)
}

// ConfigFilter represents the linter configuration file
type ConfigFilter struct {
	Issues Processors `yaml:"issues"`
}

// Processors represents the list of processors in the configuration file
type Processors struct {
	ExcludePatterns []string `yaml:"exclude"`
}

// LoadConfigFilter reads the config file and returns a ConfigFilter struct
func LoadConfigFilter(configPath string) (*ConfigFilter, error) {
	yamlFile, err := os.ReadFile(configPath)
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
func NewFilter(config ConfigFilter) (*Filter, error) {
	var filters []Processor
	for _, pattern := range config.Issues.ExcludePatterns {
		exclude := NewExclude(pattern)
		filters = append(filters, *exclude)
	}

	runner := Filter{
		processors: filters,
	}

	return &runner, nil
}
