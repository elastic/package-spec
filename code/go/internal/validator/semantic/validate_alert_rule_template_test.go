// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package semantic

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/elastic/package-spec/v3/code/go/internal/fspath"
)

const validV2Template = `{
  "id": "test-v2",
  "type": "alerting_rule_template",
  "managed": true,
  "attributes": {
    "engine": "v2",
    "rule": {
      "kind": "alert",
      "recovery": {"strategy": "no_breach"},
      "no_data": {"strategy": "ignore"},
      "metadata": {"name": "Test", "description": "A test rule", "tags": ["Test"]},
      "schedule": {"every": "1m", "lookback": "5m"},
      "time_field": "@timestamp",
      "query": {
        "base": "FROM metrics | STATS value = MAX(value) BY host.name",
        "breach": {"segment": "WHERE value > 1"}
      },
      "state_transition": {"pending": {"count": 2}, "recovering": {"count": 0}},
      "grouping": {"fields": ["host.name"]},
      "artifacts": [{"id": "test-v2-runbook", "type": "runbook", "data": {"content": "Investigate."}}]
    }
  }
}`

const validSignalTemplate = `{
  "id": "test-signal",
  "type": "alerting_rule_template",
  "attributes": {
    "engine": "v2",
    "rule": {
      "kind": "signal",
      "metadata": {"name": "Signal"},
      "schedule": {"every": "5s"},
      "query": {"base": "FROM logs | WHERE event.action == \"start\""}
    }
  }
}`

func TestV2TemplateStructure(t *testing.T) {
	require.NoError(t, validateAlertRuleTemplateJSON([]byte(validV2Template)))
	require.NoError(t, validateAlertRuleTemplateJSON([]byte(validSignalTemplate)))
	require.NoError(t, validateAlertRuleTemplateJSON([]byte(`{"attributes":{"name":"Classic"}}`)))
	require.NoError(t, validateAlertRuleTemplateJSON([]byte(`{"attributes":{"engine":"v1","name":"Classic"}}`)))
	// Classic templates are skipped even when engine is null or non-string, or
	// when classic-only fields like id/type are not strings. Only an explicit
	// "v2" engine triggers structural validation.
	require.NoError(t, validateAlertRuleTemplateJSON([]byte(`{"attributes":{"engine":null,"name":"Classic"}}`)))
	require.NoError(t, validateAlertRuleTemplateJSON([]byte(`{"id":123,"type":false,"attributes":{"name":"Classic"}}`)))
	require.NoError(t, validateAlertRuleTemplateJSON([]byte(`{"attributes":{"engine":7,"name":"Classic"}}`)))
	require.NoError(t, validateAlertRuleTemplateJSON([]byte(`{"id": "some-rule",`)))
	require.NoError(t, validateAlertRuleTemplateJSON([]byte(`not json`)))
	require.NoError(t, validateAlertRuleTemplateJSON([]byte(`{"id":"future","type":"alerting_rule_template","attributes":{"engine":"v2","future":true,"rule":{"kind":"alert","future_option":true,"metadata":{"name":"Future"},"schedule":{"every":"1m"},"query":{"base":"FROM logs"},"recovery":{"strategy":"no_breach"},"no_data":{"strategy":"ignore"}}}}`)))

	rule := func(doc map[string]any) map[string]any {
		return doc["attributes"].(map[string]any)["rule"].(map[string]any)
	}
	tests := []struct {
		name string
		edit func(map[string]any)
		want string
	}{
		{"kind", func(doc map[string]any) { rule(doc)["kind"] = "not-a-valid-rule-kind" }, "rule.kind"},
		{"metadata", func(doc map[string]any) { delete(rule(doc), "metadata") }, "metadata"},
		{"schedule", func(doc map[string]any) { delete(rule(doc), "schedule") }, "schedule"},
		{"duration", func(doc map[string]any) { rule(doc)["schedule"] = map[string]any{"every": "yesterday"} }, "schedule.every"},
		{"schedule below minimum", func(doc map[string]any) { rule(doc)["schedule"] = map[string]any{"every": "1s"} }, "schedule.every"},
		{"query", func(doc map[string]any) { delete(rule(doc), "query") }, "rule.query"},
		{"removed format", func(doc map[string]any) {
			rule(doc)["query"].(map[string]any)["format"] = "composed"
		}, "query.format"},
		{"base", func(doc map[string]any) { rule(doc)["query"] = map[string]any{"base": " "} }, "query.base"},
		{"breach query field", func(doc map[string]any) {
			rule(doc)["query"].(map[string]any)["breach"] = map[string]any{"query": "ROW x=1"}
		}, "use segment"},
		{"missing recovery", func(doc map[string]any) { delete(rule(doc), "recovery") }, "require recovery"},
		{"missing no_data", func(doc map[string]any) { delete(rule(doc), "no_data") }, "require no_data"},
		{"condition without breach", func(doc map[string]any) {
			query := rule(doc)["query"].(map[string]any)
			delete(query, "breach")
			rule(doc)["recovery"] = map[string]any{"strategy": "condition", "segment": "WHERE value < 1"}
		}, "requires query.breach"},
		{"manual recovering", func(doc map[string]any) {
			rule(doc)["recovery"] = map[string]any{"strategy": "manual"}
		}, "cannot set state_transition.recovering"},
		{"signal recovery", func(doc map[string]any) {
			rule(doc)["kind"] = "signal"
			delete(rule(doc), "state_transition")
		}, "signal rules cannot set recovery"},
		{"negative delay", func(doc map[string]any) {
			rule(doc)["state_transition"] = map[string]any{"pending": map[string]any{"count": -1}}
		}, "must be an integer"},
		{"fractional delay", func(doc map[string]any) {
			rule(doc)["state_transition"] = map[string]any{"pending": map[string]any{"count": 1.5}}
		}, "must be an integer"},
		{"group shape", func(doc map[string]any) { rule(doc)["grouping"] = map[string]any{"fields": "host"} }, "grouping.fields must be an array"},
		{"group missing", func(doc map[string]any) { rule(doc)["grouping"] = map[string]any{} }, "grouping.fields must be an array"},
		{"group entries", func(doc map[string]any) {
			rule(doc)["grouping"] = map[string]any{"fields": []any{1}}
		}, "grouping.fields entries must be strings"},
		{"empty field", func(doc map[string]any) {
			rule(doc)["grouping"] = map[string]any{"fields": []string{""}}
		}, "empty or oversized"},
		{"empty groups rejected", func(doc map[string]any) {
			rule(doc)["grouping"] = map[string]any{"fields": []string{}}
		}, "grouping.fields must contain 1 to"},
		{"dashboard camel case", func(doc map[string]any) {
			rule(doc)["artifacts"] = []any{map[string]any{"id": "dash", "type": "dashboard", "data": map[string]any{"dashboardId": "abc"}}}
		}, "dashboard_id"},
		{"artifacts not array", func(doc map[string]any) { rule(doc)["artifacts"] = "runbook" }, "rule.artifacts must be an array"},
		{"artifacts entries", func(doc map[string]any) { rule(doc)["artifacts"] = []any{"runbook"} }, "rule.artifacts entries must be objects"},
		{"query syntax needs stack", func(doc map[string]any) {
			rule(doc)["query"].(map[string]any)["base"] = "THIS IS NOT ESQL"
		}, ""},
		{"unsupported engine", func(doc map[string]any) { doc["attributes"].(map[string]any)["engine"] = "v3" }, "unsupported alert template engine"},
		{"numeric id", func(doc map[string]any) { doc["id"] = 5 }, "id is required"},
		{"numeric type", func(doc map[string]any) { doc["type"] = 5 }, "type must be alerting_rule_template"},
		{"routing tags", func(doc map[string]any) {
			rule(doc)["metadata"].(map[string]any)["routing_tags"] = []string{"oncall"}
		}, ""},
		{"routing tags not array", func(doc map[string]any) {
			rule(doc)["metadata"].(map[string]any)["routing_tags"] = "oncall"
		}, "rule.metadata.routing_tags must be an array of strings"},
		{"routing tags empty", func(doc map[string]any) {
			rule(doc)["metadata"].(map[string]any)["routing_tags"] = []string{}
		}, "rule.metadata.routing_tags must contain 1 to"},
		{"routing tags on signal", func(doc map[string]any) {
			rule(doc)["kind"] = "signal"
			for _, field := range []string{"recovery", "no_data", "state_transition"} {
				delete(rule(doc), field)
			}
			rule(doc)["metadata"].(map[string]any)["routing_tags"] = []string{"oncall"}
		}, "signal rules cannot set rule.metadata.routing_tags"},
		{"builder", func(doc map[string]any) {
			rule(doc)["metadata"].(map[string]any)["builder"] = map[string]any{"type": "threshold"}
		}, ""},
		{"builder not object", func(doc map[string]any) {
			rule(doc)["metadata"].(map[string]any)["builder"] = "threshold"
		}, "rule.metadata.builder must be an object"},
		{"builder blank type", func(doc map[string]any) {
			rule(doc)["metadata"].(map[string]any)["builder"] = map[string]any{"type": " "}
		}, "rule.metadata.builder must be an object"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var doc map[string]any
			require.NoError(t, json.Unmarshal([]byte(validV2Template), &doc))
			tt.edit(doc)
			raw, err := json.Marshal(doc)
			require.NoError(t, err)
			err = validateAlertRuleTemplateJSON(raw)
			if tt.want == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, tt.want)
			}
		})
	}
}

func TestV2TemplateStringLimits(t *testing.T) {
	fields := []struct {
		name  string
		limit int
		set   func(map[string]any, string)
	}{
		{"id", 150, func(doc map[string]any, value string) { doc["id"] = value }},
		{"name", 256, func(doc map[string]any, value string) { templateRule(doc)["metadata"].(map[string]any)["name"] = value }},
		{"description", 1024, func(doc map[string]any, value string) {
			templateRule(doc)["metadata"].(map[string]any)["description"] = value
		}},
		{"tag", 128, func(doc map[string]any, value string) {
			templateRule(doc)["metadata"].(map[string]any)["tags"] = []string{value}
		}},
		{"routing tag", 128, func(doc map[string]any, value string) {
			templateRule(doc)["metadata"].(map[string]any)["routing_tags"] = []string{value}
		}},
		{"builder type", 64, func(doc map[string]any, value string) {
			templateRule(doc)["metadata"].(map[string]any)["builder"] = map[string]any{"type": value}
		}},
		{"time_field", 256, func(doc map[string]any, value string) { templateRule(doc)["time_field"] = value }},
		{"base", 10000, func(doc map[string]any, value string) { templateRule(doc)["query"].(map[string]any)["base"] = value }},
		{"breach", 10000, func(doc map[string]any, value string) {
			templateRule(doc)["query"].(map[string]any)["breach"] = map[string]any{"segment": value}
		}},
		{"grouping", 256, func(doc map[string]any, value string) {
			templateRule(doc)["grouping"] = map[string]any{"fields": []string{value}}
		}},
		{"runbook", 50000, func(doc map[string]any, value string) {
			templateRule(doc)["artifacts"] = []any{map[string]any{"id": "runbook", "type": "runbook", "data": map[string]any{"content": value}}}
		}},
		{"dashboard", 1024, func(doc map[string]any, value string) {
			templateRule(doc)["artifacts"] = []any{map[string]any{"id": "dashboard", "type": "dashboard", "data": map[string]any{"dashboard_id": value}}}
		}},
		{"recovery segment", 10000, func(doc map[string]any, value string) {
			templateRule(doc)["recovery"] = map[string]any{"strategy": "condition", "segment": value}
		}},
		{"recovery query", 10000, func(doc map[string]any, value string) {
			templateRule(doc)["recovery"] = map[string]any{"strategy": "query", "query": value}
		}},
		{"artifact id", 150, func(doc map[string]any, value string) {
			templateRule(doc)["artifacts"] = []any{map[string]any{"id": value, "type": "custom", "data": map[string]any{}}}
		}},
		{"artifact type", 128, func(doc map[string]any, value string) {
			templateRule(doc)["artifacts"] = []any{map[string]any{"id": "custom", "type": value, "data": map[string]any{}}}
		}},
		{"artifact data key", 256, func(doc map[string]any, value string) {
			templateRule(doc)["artifacts"] = []any{map[string]any{"id": "custom", "type": "custom", "data": map[string]any{value: true}}}
		}},
	}
	for _, field := range fields {
		for _, character := range []string{"a", "é", "😀"} {
			// JavaScript string limits count an astral character as two UTF-16 units.
			count := field.limit
			if character == "😀" {
				count /= 2
			}
			t.Run(field.name+"/"+character, func(t *testing.T) {
				var doc map[string]any
				require.NoError(t, json.Unmarshal([]byte(validV2Template), &doc))
				field.set(doc, strings.Repeat(character, count))
				raw, err := json.Marshal(doc)
				require.NoError(t, err)
				require.NoError(t, validateAlertRuleTemplateJSON(raw))
				field.set(doc, strings.Repeat(character, count+1))
				raw, err = json.Marshal(doc)
				require.NoError(t, err)
				require.Error(t, validateAlertRuleTemplateJSON(raw))
			})
		}
	}
}

func TestV2TemplateDurationBounds(t *testing.T) {
	for _, field := range []string{"lookback", "pending", "recovering"} {
		for _, duration := range []string{"0ms", "0s", "0m", "0h", "0d", "0w", "365d", "366d", "-1s", "invalid"} {
			t.Run(field+"/"+duration, func(t *testing.T) {
				var doc map[string]any
				require.NoError(t, json.Unmarshal([]byte(validV2Template), &doc))
				rule := templateRule(doc)
				if field == "lookback" {
					rule["schedule"].(map[string]any)[field] = duration
				} else {
					rule["state_transition"].(map[string]any)[field] = map[string]any{"timeframe": duration}
				}
				raw, err := json.Marshal(doc)
				require.NoError(t, err)
				err = validateAlertRuleTemplateJSON(raw)
				if duration == "366d" || duration == "-1s" || duration == "invalid" {
					require.Error(t, err)
				} else {
					require.NoError(t, err)
				}
			})
		}
	}
	// Kibana caps duration strings at 32 characters before parsing them.
	for _, test := range []struct {
		every string
		valid bool
	}{
		{strings.Repeat("0", 30) + "1m", true},
		{strings.Repeat("0", 31) + "1m", false},
	} {
		var doc map[string]any
		require.NoError(t, json.Unmarshal([]byte(validV2Template), &doc))
		templateRule(doc)["schedule"].(map[string]any)["every"] = test.every
		raw, err := json.Marshal(doc)
		require.NoError(t, err)
		if test.valid {
			require.NoError(t, validateAlertRuleTemplateJSON(raw))
		} else {
			require.ErrorContains(t, validateAlertRuleTemplateJSON(raw), "schedule.every")
		}
	}

	var doc map[string]any
	require.NoError(t, json.Unmarshal([]byte(validV2Template), &doc))
	templateRule(doc)["schedule"].(map[string]any)["every"] = "0s"
	raw, err := json.Marshal(doc)
	require.NoError(t, err)
	require.ErrorContains(t, validateAlertRuleTemplateJSON(raw), "schedule.every")
}

func TestV2TemplateNoDataQuery(t *testing.T) {
	queries := []struct {
		name  string
		value any
		valid bool
	}{
		{"query", "FROM metrics", true},
		{"syntax needs stack", "THIS IS NOT ESQL", true},
		{"unicode at limit", strings.Repeat("é", 10000), true},
		{"oversized", strings.Repeat("a", 10001), false},
		{"empty", "", false},
		{"whitespace", " \n ", false},
		{"null", nil, false},
		{"object", map[string]any{}, false},
		{"array", []string{}, false},
		{"number", 1, false},
	}
	for _, strategy := range []string{"keep_last", "resolve"} {
		for _, hasBreach := range []bool{false, true} {
			for _, query := range queries {
				t.Run(fmt.Sprintf("%s/breach=%t/%s", strategy, hasBreach, query.name), func(t *testing.T) {
					var doc map[string]any
					require.NoError(t, json.Unmarshal([]byte(validV2Template), &doc))
					rule := templateRule(doc)
					if !hasBreach {
						delete(rule["query"].(map[string]any), "breach")
					}
					rule["no_data"] = map[string]any{"strategy": strategy, "query": query.value}
					raw, err := json.Marshal(doc)
					require.NoError(t, err)
					err = validateAlertRuleTemplateJSON(raw)
					if query.valid {
						require.NoError(t, err)
					} else {
						require.ErrorContains(t, err, "no_data.query")
					}
				})
			}
		}
	}
}

func TestV2TemplateTransitionCounts(t *testing.T) {
	for _, phase := range []string{"pending", "recovering"} {
		for _, count := range []string{"0", "-0", "2", "2.0", "2e0", "2e1", "1000.0", "-1", "1.5", "1e-1", "1001", "1e999", `"2"`, "true", "[]", "{}"} {
			t.Run(phase+"/"+count, func(t *testing.T) {
				var doc map[string]any
				require.NoError(t, json.Unmarshal([]byte(validV2Template), &doc))
				// Keep the original JSON spelling; decoding and re-encoding a float
				// would otherwise normalize 2.0 and 2e0 to 2 before validation.
				templateRule(doc)["state_transition"].(map[string]any)[phase] = map[string]any{"count": json.RawMessage(count)}
				raw, err := json.Marshal(doc)
				require.NoError(t, err)
				err = validateAlertRuleTemplateJSON(raw)
				switch count {
				case "0", "-0", "2", "2.0", "2e0", "2e1", "1000.0":
					require.NoError(t, err)
				default:
					require.ErrorContains(t, err, "must be an integer")
				}
			})
		}
	}
}

func templateRule(doc map[string]any) map[string]any {
	return doc["attributes"].(map[string]any)["rule"].(map[string]any)
}

func TestV2TemplateOptionalNulls(t *testing.T) {
	paths := [][]string{
		{"metadata", "description"}, {"metadata", "tags"},
		{"metadata", "routing_tags"}, {"metadata", "builder"},
		{"schedule", "lookback"}, {"time_field"}, {"query", "breach"},
		{"query", "format"}, {"grouping"}, {"artifacts"},
		{"state_transition", "pending"}, {"state_transition", "recovering"},
		{"state_transition", "pending", "count"}, {"state_transition", "pending", "timeframe"},
		{"state_transition", "pending", "operator"},
		{"state_transition", "recovering", "count"}, {"state_transition", "recovering", "timeframe"},
		{"state_transition", "recovering", "operator"},
		{"recovery", "query"}, {"recovery", "segment"}, {"no_data", "query"},
	}
	for _, path := range paths {
		t.Run(strings.Join(path, "."), func(t *testing.T) {
			var doc map[string]any
			require.NoError(t, json.Unmarshal([]byte(validV2Template), &doc))
			rule := templateRule(doc)
			// Each phase remains valid when either threshold is omitted.
			for _, phase := range []string{"pending", "recovering"} {
				rule["state_transition"].(map[string]any)[phase].(map[string]any)["timeframe"] = "1m"
			}
			parent := rule
			for _, field := range path[:len(path)-1] {
				parent = parent[field].(map[string]any)
			}
			field := path[len(path)-1]
			delete(parent, field)
			raw, err := json.Marshal(doc)
			require.NoError(t, err)
			require.NoError(t, validateAlertRuleTemplateJSON(raw), "omission should be accepted")
			parent[field] = nil
			raw, err = json.Marshal(doc)
			require.NoError(t, err)
			require.Error(t, validateAlertRuleTemplateJSON(raw), "explicit null should be rejected")
		})
	}
	for _, template := range []string{validV2Template, validSignalTemplate} {
		var doc map[string]any
		require.NoError(t, json.Unmarshal([]byte(template), &doc))
		templateRule(doc)["state_transition"] = nil
		raw, err := json.Marshal(doc)
		require.NoError(t, err)
		require.ErrorContains(t, validateAlertRuleTemplateJSON(raw), "state_transition")
	}
}

func TestV2SignalTransitions(t *testing.T) {
	for _, transition := range []string{"missing", "null", "{}", `{"pending":{"count":0}}`, "[]", "2", "false", `"invalid"`} {
		t.Run(transition, func(t *testing.T) {
			var doc map[string]any
			require.NoError(t, json.Unmarshal([]byte(validSignalTemplate), &doc))
			if transition != "missing" {
				templateRule(doc)["state_transition"] = json.RawMessage(transition)
			}
			raw, err := json.Marshal(doc)
			require.NoError(t, err)
			err = validateAlertRuleTemplateJSON(raw)
			if transition == "missing" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, "signal rules cannot set state_transition")
			}
		})
	}
	for _, field := range []string{"recovery", "no_data"} {
		t.Run(field+"/null", func(t *testing.T) {
			var doc map[string]any
			require.NoError(t, json.Unmarshal([]byte(validSignalTemplate), &doc))
			templateRule(doc)[field] = nil
			raw, err := json.Marshal(doc)
			require.NoError(t, err)
			require.ErrorContains(t, validateAlertRuleTemplateJSON(raw), "signal rules cannot set recovery or no_data")
		})
	}
}

func TestAlertRuleTemplateSemanticRule(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "kibana", "alerting_rule_template")
	require.NoError(t, os.MkdirAll(dir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "valid.json"), []byte(validV2Template), 0o644))
	require.Empty(t, ValidateAlertRuleTemplates(fspath.DirFS(root)))

	require.NoError(t, os.WriteFile(filepath.Join(dir, "invalid.json"), []byte(`{"id":"invalid","type":"alerting_rule_template","attributes":{"engine":"v2","rule":{"kind":"invalid"}}}`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "truncated.json"), []byte(`{"id": "truncated",`), 0o644))
	errs := ValidateAlertRuleTemplates(fspath.DirFS(root))
	require.Len(t, errs, 1)
	require.ErrorContains(t, errs[0], "rule.kind")
	require.ErrorContains(t, errs[0], "SVR00014")
	require.ErrorContains(t, errs[0], filepath.Join(dir, "invalid.json"))
}

func TestParseAlertRuleTemplate(t *testing.T) {
	for _, test := range []struct {
		name string
		data string
		skip bool
		want string
	}{
		{"v2", validV2Template, false, ""},
		{"invalid v2 id and type", `{"id":5,"type":false,"attributes":{"engine":"v2"}}`, false, ""},
		{"missing engine", `{"id":5,"type":false,"attributes":{}}`, true, ""},
		{"null engine", `{"attributes":{"engine":null}}`, true, ""},
		{"numeric engine", `{"attributes":{"engine":7}}`, true, ""},
		{"empty engine", `{"attributes":{"engine":""}}`, true, ""},
		{"classic engine", `{"id":5,"attributes":{"engine":"v1"}}`, true, ""},
		{"non-object attributes", `{"attributes":[]}`, true, ""},
		{"null attributes", `{"attributes":null}`, true, ""},
		{"invalid JSON", `{"attributes":{"engine":"v2"`, true, ""},
		{"unsupported engine", `{"attributes":{"engine":"v3"}}`, false, `unsupported alert template engine "v3"`},
		{"uppercase engine", `{"attributes":{"engine":"V2"}}`, false, `unsupported alert template engine "V2"`},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, skip, err := parseAlertRuleTemplate([]byte(test.data))
			require.Equal(t, test.skip, skip)
			if test.want == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, test.want)
			}
		})
	}
}

func TestAlertRuleTemplatesV2FormatVersion(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "kibana", "alerting_rule_template")
	require.NoError(t, os.MkdirAll(dir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "classic.json"), []byte(`{"id":"classic","type":"alerting_rule_template","attributes":{"engine":"v1","name":"Classic"}}`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "truncated.json"), []byte(`{"id": "truncated",`), 0o644))
	require.Empty(t, ValidateAlertRuleTemplatesV2FormatVersion(fspath.DirFS(root)))

	require.NoError(t, os.WriteFile(filepath.Join(dir, "v2.json"), []byte(`{"id":"v2","type":"alerting_rule_template","attributes":{"engine":"v2"}}`), 0o644))
	errs := ValidateAlertRuleTemplatesV2FormatVersion(fspath.DirFS(root))
	require.Len(t, errs, 1)
	require.ErrorContains(t, errs[0], filepath.Join(dir, "v2.json"))
	require.ErrorContains(t, errs[0], "format_version 3.7.0")
	require.Empty(t, errs[0].Code())

	require.NoError(t, os.WriteFile(filepath.Join(dir, "v3.json"), []byte(`{"id":"v3","type":"alerting_rule_template","attributes":{"engine":"v3"}}`), 0o644))
	errs = ValidateAlertRuleTemplatesV2FormatVersion(fspath.DirFS(root))
	require.Len(t, errs, 2)
	require.ErrorContains(t, errs[1], filepath.Join(dir, "v3.json"))
	require.ErrorContains(t, errs[1], `unsupported alert template engine "v3"`)
	require.Empty(t, errs[1].Code())
}

func TestV2TemplateNonemptyCollections(t *testing.T) {
	for _, test := range []struct {
		name string
		edit func(map[string]any)
		want string
	}{
		{"blank description", func(rule map[string]any) { rule["metadata"].(map[string]any)["description"] = "   " }, "metadata.description"},
		{"empty artifacts", func(rule map[string]any) { rule["artifacts"] = []any{} }, "artifacts must contain 1"},
	} {
		t.Run(test.name, func(t *testing.T) {
			var doc map[string]any
			require.NoError(t, json.Unmarshal([]byte(validV2Template), &doc))
			test.edit(templateRule(doc))
			raw, err := json.Marshal(doc)
			require.NoError(t, err)
			require.ErrorContains(t, validateAlertRuleTemplateJSON(raw), test.want)
		})
	}
}

func TestV2TemplateUntrimmedStrings(t *testing.T) {
	// Kibana checks these with min(1) and no trim, so whitespace is a valid value.
	for _, test := range []struct {
		name string
		edit func(map[string]any)
	}{
		{"id", func(doc map[string]any) { doc["id"] = " " }},
		{"name", func(doc map[string]any) { templateRule(doc)["metadata"].(map[string]any)["name"] = " " }},
		{"tag", func(doc map[string]any) { templateRule(doc)["metadata"].(map[string]any)["tags"] = []string{" "} }},
		{"time_field", func(doc map[string]any) { templateRule(doc)["time_field"] = " " }},
		{"grouping", func(doc map[string]any) { templateRule(doc)["grouping"] = map[string]any{"fields": []string{" "}} }},
		{"artifact", func(doc map[string]any) {
			templateRule(doc)["artifacts"] = []any{map[string]any{"id": " ", "type": " ", "data": map[string]any{" ": true}}}
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			var doc map[string]any
			require.NoError(t, json.Unmarshal([]byte(validV2Template), &doc))
			test.edit(doc)
			raw, err := json.Marshal(doc)
			require.NoError(t, err)
			require.NoError(t, validateAlertRuleTemplateJSON(raw))
		})
	}
}

func TestV2TemplateArtifactData(t *testing.T) {
	fields := func(count int) map[string]any {
		data := map[string]any{}
		for i := range count {
			data[fmt.Sprintf("field%d", i)] = true
		}
		return data
	}
	for _, test := range []struct {
		name string
		data map[string]any
		want string
	}{
		{"field limit", fields(32), ""},
		{"too many fields", fields(33), "rule.artifacts[0].data must have at most 32 fields"},
		{"empty key", map[string]any{"": true}, "rule.artifacts[0].data keys must be 1 to 256 characters"},
		// {"value":"…"} adds 12 characters around the string.
		{"serialized limit", map[string]any{"value": strings.Repeat("é", 131072-12)}, ""},
		{"serialized too long", map[string]any{"value": strings.Repeat("é", 131072-11)}, "must not exceed 131072 characters when serialized"},
	} {
		t.Run(test.name, func(t *testing.T) {
			var doc map[string]any
			require.NoError(t, json.Unmarshal([]byte(validV2Template), &doc))
			templateRule(doc)["artifacts"] = []any{map[string]any{"id": "custom", "type": "custom", "data": test.data}}
			raw, err := json.Marshal(doc)
			require.NoError(t, err)
			if test.want == "" {
				require.NoError(t, validateAlertRuleTemplateJSON(raw))
			} else {
				require.ErrorContains(t, validateAlertRuleTemplateJSON(raw), test.want)
			}
		})
	}
}

func TestV2TemplateJavaScriptWhitespace(t *testing.T) {
	// Kibana trims with String.prototype.trim, which strips U+FEFF but not U+0085.
	for _, test := range []struct {
		name  string
		value string
		want  string
	}{
		{"next line is not whitespace", "\u0085", ""},
		{"byte order mark is whitespace", "\uFEFF", "metadata.description"},
		{"no-break space is whitespace", "\u00A0\u2028", "metadata.description"},
	} {
		t.Run(test.name, func(t *testing.T) {
			var doc map[string]any
			require.NoError(t, json.Unmarshal([]byte(validV2Template), &doc))
			templateRule(doc)["metadata"].(map[string]any)["description"] = test.value
			raw, err := json.Marshal(doc)
			require.NoError(t, err)
			if test.want == "" {
				require.NoError(t, validateAlertRuleTemplateJSON(raw))
			} else {
				require.ErrorContains(t, validateAlertRuleTemplateJSON(raw), test.want)
			}
		})
	}

	var doc map[string]any
	require.NoError(t, json.Unmarshal([]byte(validV2Template), &doc))
	templateRule(doc)["query"].(map[string]any)["breach"] = map[string]any{"segment": "\uFEFF"}
	raw, err := json.Marshal(doc)
	require.NoError(t, err)
	require.ErrorContains(t, validateAlertRuleTemplateJSON(raw), "rule.query.breach.segment is required")
}

func TestTemplateJSONLength(t *testing.T) {
	// Expected values are JSON.stringify(JSON.parse(raw)).length from Node.js.
	for raw, want := range map[string]int{
		`{"v":"\u2028"}`:           9,
		`{"n":1e21}`:               11,
		`{"n":1.000}`:              7,
		`{"n":1e20}`:               27,
		`{"s":"\ud800"}`:           14,
		`{"s":"\ud83d\ude00"}`:     10,
		`{"a":1,"a":2}`:            7,
		`{"s":"a\u0001\n\"\/"}`:    20,
		`[1, true, null, -0.5e-7]`: 19,
		`{"n":1e400}`:              10,
		`{"n":0.000001}`:           14,
		`{"n":1e-7}`:               10,
		`{"n":-0}`:                 7,
		`{"n":123.456e5}`:          14,
		`{"n":5e-324}`:             12,
		`{ "k\tey" : "é" }`:        13,
	} {
		require.Equal(t, want, templateJSONLength(json.RawMessage(raw)), raw)
	}
}

func TestV2TemplateReportsEveryArtifact(t *testing.T) {
	var doc map[string]any
	require.NoError(t, json.Unmarshal([]byte(validV2Template), &doc))
	templateRule(doc)["artifacts"] = []any{
		map[string]any{"id": "", "type": "runbook", "data": map[string]any{"content": "Investigate."}},
		map[string]any{"id": "dash", "type": "dashboard", "data": map[string]any{"dashboardId": "abc"}},
	}
	raw, err := json.Marshal(doc)
	require.NoError(t, err)
	err = validateAlertRuleTemplateJSON(raw)
	require.ErrorContains(t, err, "rule.artifacts[0].id is required")
	require.ErrorContains(t, err, "dashboard artifact data.dashboardId is not accepted")
}

// validateAlertRuleTemplateJSON exercises the shared engine parser and the v2
// structural checks with the same inline documents used by these unit tests.
func validateAlertRuleTemplateJSON(data []byte) error {
	template, skip, err := parseAlertRuleTemplate(data)
	if err != nil || skip {
		return err
	}
	return validateAlertRuleTemplate(template)
}
