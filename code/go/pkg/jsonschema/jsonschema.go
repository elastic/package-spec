// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package jsonschema

import (
	"fmt"
	"log"

	"github.com/Masterminds/semver/v3"
	"github.com/pkg/errors"

	"github.com/elastic/package-spec/v2/code/go/internal/spec"
)

func AllJSONSchemas(version, pkgType string) error {
	fmt.Printf("All json schemas (%s - %s)\n", pkgType, version)
	specVersion, err := semver.NewVersion(version)
	if err != nil {
		return err
	}

	rootSpec, err := spec.NewSpec(*specVersion)
	if err != nil {
		return err
	}

	rendered, err := rootSpec.RenderAllJsonSchema(pkgType)

	for _, itemSpec := range rendered {
		fmt.Printf("Name: %s\n", itemSpec.Name)
		fmt.Printf("Content:\n%s\n", itemSpec.JSONSchema)
	}
	return nil
}

func JSONSchema(itemPath, version, pkgType string) ([]byte, error) {
	specVersion, err := semver.NewVersion(version)
	if err != nil {
		return nil, err
	}

	rootSpec, err := spec.NewSpec(*specVersion)
	if err != nil {
		return nil, errors.Wrap(err, "invalid package spec version")
	}

	rendered, err := rootSpec.RenderJsonSchema(itemPath, pkgType)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to render jsonschema for %s", itemPath)
	}

	log.Printf("Rendered jsonschema for path: %s\n", itemPath)
	return rendered.JSONSchema, nil
}
