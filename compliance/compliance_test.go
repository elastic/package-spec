// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package main

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cucumber/godog"
	messages "github.com/cucumber/messages/go/v21"
	"golang.org/x/exp/slices"
	"gopkg.in/yaml.v3"
)

//go:embed features/*
var featuresFS embed.FS

func TestSpecCompliance(t *testing.T) {
	paths := []string{"features"}
	if pathsEnv := os.Getenv("TEST_SPEC_FEATURES"); pathsEnv != "" {
		paths = strings.Split(pathsEnv, ",")
	}
	checkFeaturesVersions(t, featuresFS, paths)

	junitFileName := os.Getenv("TEST_SPEC_JUNIT")
	if junitFileName == "" {
		junitFileName = "report.xml"
	}

	suite := godog.TestSuite{
		ScenarioInitializer: InitializeScenario,
		Options: &godog.Options{
			Format:   fmt.Sprintf("pretty,junit:%s", junitFileName),
			Paths:    paths,
			Tags:     versionsToTest(t),
			FS:       featuresFS,
			TestingT: t,
			Strict:   false,
		},
	}

	if suite.Run() != 0 {
		t.Fatalf("target deployment doesn't comply with Package Spec %s", versionToComply(t))
	}
}

func indexTemplateHasAFieldWith(indexTemplateName, fieldName, condition string) error {
	es, err := NewElasticsearchClient()
	if err != nil {
		return err
	}

	indexTemplate, err := es.SimulateIndexTemplate(indexTemplateName)
	if err != nil {
		return err
	}

	fieldMapping, err := indexTemplate.FieldMapping(fieldName)
	if err != nil {
		return err
	}

	if fieldMapping.CheckCondition(condition) {
		return nil
	}

	d, err := json.MarshalIndent(fieldMapping, "", "  ")
	if err != nil {
		return err
	}
	fmt.Printf("Found the following mapping for field %q:\n", fieldName)
	fmt.Println(string(d))
	return fmt.Errorf("condition %q not satisfied by field %q", condition, fieldName)
}

func thePackageIsInstalled(packageName string) error {
	packagePath, err := findTestPackage(packageName)
	if err != nil {
		return err
	}

	elasticPackage, err := NewElasticPackage()
	if err != nil {
		return err
	}
	defer elasticPackage.Close()

	err = elasticPackage.Install(packagePath)
	if err != nil {
		return fmt.Errorf("cannot install package %q: %w", packagePath, err)
	}

	return nil
}

func findTestPackage(packageName string) (string, error) {
	// TODO: embed sample packages, so we can build a standalone test binary.
	paths := []string{
		filepath.Join("testdata", "packages", packageName),

		// Support testing with packages from the spec to avoid duplicating packages.
		filepath.Join("..", "test", "packages", packageName),
	}

	for _, path := range paths {
		info, err := os.Stat(path)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return "", fmt.Errorf("failed to check if path %s exists: %w", path, err)
		}
		if !info.IsDir() {
			return "", fmt.Errorf("\"%s\" should be a directory", path)
		}

		return path, nil
	}

	return "", godog.ErrPending
}

type contextKey string

const agentPolicyIDKey contextKey = "agentPolicyID"

func aPolicyIsCreatedWithPackage(ctx context.Context, packageName string) (context.Context, error) {
	const version = "1.0.0"
	return aPolicyIsCreatedWithPackageAndVersion(ctx, packageName, version)
}

func aPolicyIsCreatedWithPackageAndVersion(ctx context.Context, packageName, packageVersion string) (context.Context, error) {
	kibana, err := NewKibanaClient()
	if err != nil {
		return ctx, err
	}
	agentPolicyID, err := kibana.createPolicyForPackage(packageName, packageVersion)
	if err != nil {
		return ctx, err
	}
	return context.WithValue(ctx, agentPolicyIDKey, agentPolicyID), nil
}

func aPolicyIsCreatedWithPackageInputAndDataset(ctx context.Context, packageName, packageVersion, templateName, inputName, inputType, dataset string) (context.Context, error) {
	kibana, err := NewKibanaClient()
	if err != nil {
		return ctx, err
	}
	agentPolicyID, err := kibana.createPolicyForPackageInputAndDataset(packageName, packageVersion, templateName, inputName, inputType, dataset)
	if err != nil {
		return ctx, err
	}
	return context.WithValue(ctx, agentPolicyIDKey, agentPolicyID), nil
}

func thereIsAnIndexTemplateWithPattern(indexTemplateName, pattern string) error {
	es, err := NewElasticsearchClient()
	if err != nil {
		return err
	}

	indexTemplate, err := es.IndexTemplate(indexTemplateName)
	if err != nil {
		return err
	}

	if !slices.Contains(indexTemplate.IndexPatterns, pattern) {
		return fmt.Errorf("pattern %q not found in %s", pattern, indexTemplate.IndexPatterns)
	}

	return nil
}

func thereIsATransform(transformID string) error {
	es, err := NewElasticsearchClient()
	if err != nil {
		return err
	}

	resp, err := es.client.TransformGetTransform(
		es.client.TransformGetTransform.WithContext(context.TODO()),
		es.client.TransformGetTransform.WithTransformID(transformID),
	)
	if err != nil {
		return fmt.Errorf("failed to get transform %q: %w", transformID, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("transform %q not found", transformID)
	}

	return nil
}

// theTransformHasAliasConfigured checks that the alias is present in the
// transform's dest.aliases configuration. Actual alias creation on the
// destination index can only be tested with real source data, because
// Elasticsearch applies dest.aliases only when the destination index is
// created by the transform indexer.
func theTransformHasAliasConfigured(transformID, aliasName string) error {
	es, err := NewElasticsearchClient()
	if err != nil {
		return err
	}
	return es.TransformHasAlias(transformID, aliasName)
}

func indexTemplateIsConfiguredFor(indexTemplateName, option string) error {
	es, err := NewElasticsearchClient()
	if err != nil {
		return err
	}

	indexTemplate, err := es.SimulateIndexTemplate(indexTemplateName)
	if err != nil {
		return err
	}

	switch option {
	case "synthetic source mode":
		if indexTemplate.Settings.Index.Mapping.Source.Mode == "synthetic" {
			return nil
		}
		if indexTemplate.Mappings.Source.Mode == "synthetic" {
			return nil
		}
		return errors.New("synthetic source mode is not enabled")

	case "lookup index mode":
		if indexTemplate.Settings.Index.Mode == "lookup" {
			return nil
		}
		return errors.New("lookup mode is not enabled")

	case "time_series index mode":
		if indexTemplate.Settings.Index.Mode == "time_series" {
			return nil
		}
		return errors.New("time_series mode is not enabled")
	}

	return godog.ErrPending
}

func thereIsAnSloTemplate(templateID string) error {
	kibana, err := NewKibanaClient()
	if err != nil {
		return err
	}
	return kibana.mustExistSLOTemplate(templateID)
}

func thereIsADashboard(dashboardID string) error {
	kibana, err := NewKibanaClient()
	if err != nil {
		return err
	}
	err = kibana.mustExistDashboard(dashboardID)
	if err != nil {
		return err
	}
	return nil
}

func thereIsADetectionRule(detectionRuleID string) error {
	kibana, err := NewKibanaClient()
	if err != nil {
		return err
	}
	err = kibana.mustExistDetectionRule(detectionRuleID)
	if err != nil {
		return err
	}
	return nil
}

func prebuiltDetectionRulesAreLoaded() error {
	kibana, err := NewKibanaClient()
	if err != nil {
		return err
	}
	err = kibana.loadPrebuiltDetectionRules()
	if err != nil {
		return err
	}
	return nil
}

func thereIsASecurityAIPrompt(promptID string) error {
	kibana, err := NewKibanaClient()
	if err != nil {
		return err
	}
	err = kibana.mustExistSavedObject("security-ai-prompt", promptID)
	if err != nil {
		return err
	}
	return nil
}

func theContentPackagesRequireAreInstalled(packageName string) error {
	packagePath, err := findTestPackage(packageName)
	if err != nil {
		return err
	}

	manifestData, err := os.ReadFile(filepath.Join(packagePath, "manifest.yml"))
	if err != nil {
		return fmt.Errorf("failed to read manifest for package %q: %w", packageName, err)
	}

	var manifest struct {
		Requires struct {
			Content []struct {
				Package string `yaml:"package"`
			} `yaml:"content"`
		} `yaml:"requires"`
	}
	if err := yaml.Unmarshal(manifestData, &manifest); err != nil {
		return fmt.Errorf("failed to parse manifest for package %q: %w", packageName, err)
	}

	kibana, err := NewKibanaClient()
	if err != nil {
		return err
	}

	for _, dep := range manifest.Requires.Content {
		if err := kibana.isPackageInstalled(dep.Package); err != nil {
			return fmt.Errorf("required content package %q is not installed: %w", dep.Package, err)
		}
	}

	return nil
}

func theCompiledPolicyHasDataset(ctx context.Context, expectedDataset, inputType string) error {
	agentPolicyID, ok := ctx.Value(agentPolicyIDKey).(string)
	if !ok || agentPolicyID == "" {
		return fmt.Errorf("agent policy ID not found in context; ensure a policy creation step ran first")
	}

	kibana, err := NewKibanaClient()
	if err != nil {
		return err
	}

	policy, err := kibana.getFullAgentPolicy(agentPolicyID)
	if err != nil {
		return fmt.Errorf("failed to get full agent policy: %w", err)
	}

	if inputType == "otelcol" {
		return checkDatasetInOTelPolicy(policy, expectedDataset)
	}
	return checkDatasetInInputs(policy, expectedDataset)
}

func checkDatasetInInputs(policy *fullAgentPolicy, expectedDataset string) error {
	for _, input := range policy.Inputs {
		for _, stream := range input.Streams {
			if stream.DataStream.Dataset == expectedDataset {
				return nil
			}
		}
		if input.DataStream.Dataset == expectedDataset {
			return nil
		}
	}

	d, _ := json.MarshalIndent(policy.Inputs, "", "  ")
	return fmt.Errorf("dataset %q not found in compiled policy inputs:\n%s", expectedDataset, string(d))
}

func checkDatasetInOTelPolicy(policy *fullAgentPolicy, expectedDataset string) error {
	expectedStatement := fmt.Sprintf(`set(attributes["data_stream.dataset"], "%s")`, expectedDataset)

	for name, processor := range policy.Processors {
		if !strings.HasPrefix(name, "transform/") {
			continue
		}
		if transformHasStatement(processor, expectedStatement) {
			return nil
		}
	}

	d, _ := json.MarshalIndent(policy.Processors, "", "  ")
	return fmt.Errorf("dataset statement %q not found in compiled policy processors:\n%s", expectedStatement, string(d))
}

func transformHasStatement(processor otelTransformProcessor, expected string) bool {
	allGroups := [][]otelStatementGroup{
		processor.LogStatements,
		processor.MetricStatements,
		processor.TraceStatements,
	}
	for _, groups := range allGroups {
		for _, group := range groups {
			for _, stmt := range group.Statements {
				if stmt == expected {
					return true
				}
			}
		}
	}
	return false
}

func InitializeScenario(ctx *godog.ScenarioContext) {
	ctx.Before(func(ctx context.Context, sc *godog.Scenario) (context.Context, error) {
		skipped := slices.ContainsFunc(sc.Tags, func(elem *messages.PickleTag) bool {
			return elem.Name == "@skip"
		})
		if skipped {
			return ctx, godog.ErrSkip
		}
		return ctx, nil
	})
	ctx.Step(`^index template "([^"]*)" has a field "([^"]*)" with "([^"]*)"$`, indexTemplateHasAFieldWith)
	ctx.Step(`^the "([^"]*)" package is installed$`, thePackageIsInstalled)
	ctx.Step(`^a policy is created with "([^"]*)" package$`, aPolicyIsCreatedWithPackage)
	ctx.Step(`^a policy is created with "([^"]*)" package and "([^"]*)" version$`, aPolicyIsCreatedWithPackageAndVersion)
	ctx.Step(`^a policy is created with "([^"]*)" package, "([^"]*)" version, "([^"]*)" template, "([^"]*)" input, "([^"]*)" input type and dataset "([^"]*)"$`, aPolicyIsCreatedWithPackageInputAndDataset)
	ctx.Step(`^there is an index template "([^"]*)" with pattern "([^"]*)"$`, thereIsAnIndexTemplateWithPattern)
	ctx.Step(`^there is a transform "([^"]*)"$`, thereIsATransform)
	ctx.Step(`^the transform "([^"]*)" has alias "([^"]*)" configured$`, theTransformHasAliasConfigured)
	ctx.Step(`^index template "([^"]*)" is configured for "([^"]*)"$`, indexTemplateIsConfiguredFor)
	ctx.Step(`^there is an SLO template "([^"]*)"$`, thereIsAnSloTemplate)
	ctx.Step(`^there is a dashboard "([^"]*)"$`, thereIsADashboard)
	ctx.Step(`^there is a detection rule "([^"]*)"$`, thereIsADetectionRule)
	ctx.Step(`^prebuilt detection rules are loaded$`, prebuiltDetectionRulesAreLoaded)
	ctx.Step(`^there is a security AI prompt "([^"]*)"$`, thereIsASecurityAIPrompt)
	ctx.Step(`^the content packages "([^"]*)" require are installed$`, theContentPackagesRequireAreInstalled)
	ctx.Step(`^the compiled policy has dataset "([^"]*)" for "([^"]*)" input type$`, theCompiledPolicyHasDataset)
}
