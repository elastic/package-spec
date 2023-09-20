// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package linter

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"

	"github.com/elastic/package-spec/v2/code/go/pkg/errors"
	"github.com/elastic/package-spec/v2/code/go/pkg/errors/processors"
)

// Runner represents the collection of processors to be applied over validation errors
type Runner struct {
	processors []processors.Processor
}

// Run runs all the processors over all the validation errors and return the filtered ones
func (r *Runner) Run(errors errors.ValidationErrors) (errors.ValidationErrors, error) {
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
func (r *Runner) AddProcessors(items []processors.Processor) {
	r.processors = append(r.processors, items...)
}

// ConfigRunner represents the linter configuration file
type ConfigRunner struct {
	Issues []Processors `yaml:"issues"`
}

// Processors represents the list of processors in the configuration file
type Processors struct {
	ExcludePatterns []string `yaml:"exclude"`
}

// LoadConfigRunner reads the config file and returns a ConfigRunner struct
func LoadConfigRunner(configPath string) (*ConfigRunner, error) {
	yamlFile, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", configPath, err)
	}
	var config ConfigRunner
	err = yaml.Unmarshal(yamlFile, &config)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config file %s: %w", configPath, err)
	}
	return &config, nil
}

// NewRunner creates a new runner given a configuration
func NewRunner(config ConfigRunner) (*Runner, error) {
	var filters []processors.Processor
	for _, p := range config.Issues {
		for _, pattern := range p.ExcludePatterns {
			exclude := processors.NewExclude(pattern)
			filters = append(filters, *exclude)
		}
	}

	runner := Runner{
		processors: filters,
	}

	return &runner, nil
}
