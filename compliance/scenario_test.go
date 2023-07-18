// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package main

import (
	"os"

	"github.com/cucumber/godog"
)

type scenarioManager struct {
	tmpDir string
}

func newScenarioManager(ctx *godog.ScenarioContext) *scenarioManager {
	sm := scenarioManager{}
	ctx.BeforeScenario(func(sc *godog.Scenario) {
		tmpDir, err := os.MkdirTemp("", "spec-compliance-***")
		if err != nil {
			panic("failed to create temporary directory: " + err.Error())
		}
		sm.tmpDir = tmpDir
	})
	ctx.AfterScenario(func(sc *godog.Scenario, err error) {
		if sm.tmpDir != "" {
			os.RemoveAll(sm.tmpDir)
		}
	})
	return &sm
}

func (sm *scenarioManager) createPackage(packageType string) error {
	switch packageType {
	case "integration":
	case "input":
	default:
		return godog.ErrPending
	}

	return nil
}

func (sm *scenarioManager) checkIndexTemplate(condition string) error {
	return godog.ErrPending
}

func (sm *scenarioManager) addToPackage(packageType string) error {
	return godog.ErrPending
}

func (sm *scenarioManager) installPackage() error {
	return godog.ErrPending
}
