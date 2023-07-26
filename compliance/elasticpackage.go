// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

const elasticPackageImportPath = "github.com/elastic/elastic-package"

type ElasticPackage struct {
	home string
}

func NewElasticPackage() (*ElasticPackage, error) {
	tmpDir, err := os.MkdirTemp("", "elastic-package-XXX")
	if err != nil {
		return nil, fmt.Errorf("failed to create configuration directory: %w", err)
	}

	return &ElasticPackage{
		home: filepath.Join(tmpDir, ".elastic-package"),
	}, nil
}

func (ep *ElasticPackage) Close() error {
	if ep.home == "" {
		return nil
	}

	return os.RemoveAll(ep.home)
}

func (ep *ElasticPackage) Install(packagePath string) error {
	cmd := exec.Command("go", "run", elasticPackageImportPath, "install")
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	cmd.Env = append(os.Environ(),
		"ELASTIC_PACKAGE_DATA_HOME="+ep.home,
		elasticPackageGetEnv("ELASTICSEARCH_HOST"),
		elasticPackageGetEnv("ELASTICSEARCH_PASSWORD"),
		elasticPackageGetEnv("ELASTICSEARCH_USERNAME"),
		elasticPackageGetEnv("KIBANA_HOST"),
	)
	cmd.Dir = packagePath
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("elastic-package failed: %w", err)
	}
	return nil
}

func elasticPackageGetEnv(name string) string {
	v := os.Getenv("ELASTIC_PACKAGE_" + name)
	if v != "" {
		return v
	}
	return os.Getenv(name)
}
