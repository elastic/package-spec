// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package validator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAlertRuleTemplateSemanticRuleEntryPoints(t *testing.T) {
	// The fixture stays on format_version 3.4.0 on purpose: alert rule template
	// structure is validated on every package format, so templates introduced in
	// older packages are covered too, not only 3.6.0 and later.
	fixture := filepath.Join("..", "..", "..", "..", "test", "packages", "good_alert_rule_templates")
	packageRoot := filepath.Join(t.TempDir(), "good_alert_rule_templates")
	require.NoError(t, os.CopyFS(packageRoot, os.DirFS(fixture)))

	templatePath := filepath.Join(packageRoot, "kibana", "alerting_rule_template", "some-rule.json")
	classic, err := os.ReadFile(templatePath)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(templatePath, []byte(strings.Replace(string(classic), `"name":`, `"engine": "v1", "name":`, 1)), 0o644))
	legacy, err := New(LegacyMode)
	require.NoError(t, err)
	require.NoError(t, legacy.ValidateFromPath(packageRoot))

	invalid := []byte(`{"id":"some-rule","type":"alerting_rule_template","attributes":{"engine":"v2","rule":{"kind":"invalid","metadata":{"name":"Bad"},"query":{"format":"composed"}}}}`)
	require.NoError(t, os.WriteFile(templatePath, invalid, 0o644))
	for _, mode := range []Mode{LegacyMode, SourceMode, BuildMode} {
		validator, err := New(mode)
		require.NoError(t, err)
		err = validator.ValidateFromPath(packageRoot)
		require.ErrorContains(t, err, "rule.kind")
		require.ErrorContains(t, err, "rule.schedule")
		require.ErrorContains(t, err, "query.format")
		require.ErrorContains(t, err, "SVR00014")
	}

	zipPath := writePackageZip(t, packageRoot, "good_alert_rule_templates")
	err = legacy.ValidateFromZip(zipPath)
	require.ErrorContains(t, err, "SVR00014")
}
