// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package semantic

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"math"
	"regexp"
	"strconv"
	"strings"

	"github.com/elastic/package-spec/v3/code/go/internal/fspath"
	"github.com/elastic/package-spec/v3/code/go/pkg/specerrors"
)

// Offline structural checks for Alerting v2 templates (Kibana 9.6 shape:
// query.base plus optional breach.segment, recovery and no_data objects, and
// nested state-transition phases). The shape and the limit constants below
// mirror Kibana's alerting-v2-schemas at commit 600a8067f05e21388a63ffcc22cf602a3ffb1aa3:
//   - rule_template_schema.ts: https://github.com/elastic/kibana/blob/600a8067f05e21388a63ffcc22cf602a3ffb1aa3/x-pack/platform/packages/shared/response-ops/alerting-v2-schemas/src/rule_template_schema.ts
//   - rule_data_schema.ts:     https://github.com/elastic/kibana/blob/600a8067f05e21388a63ffcc22cf602a3ffb1aa3/x-pack/platform/packages/shared/response-ops/alerting-v2-schemas/src/rule_data_schema.ts
//   - constants.ts:            https://github.com/elastic/kibana/blob/600a8067f05e21388a63ffcc22cf602a3ffb1aa3/x-pack/platform/packages/shared/response-ops/alerting-v2-schemas/src/constants.ts
//
// This is not Kibana's complete schema and it does not parse or execute ES|QL.
// Kibana's objects are .strict(), but this check deliberately leaves unknown
// fields in place so a later schema addition is not rejected here; a template
// can therefore pass package-spec and still be rejected by Kibana when a rule is
// created from it. Fleet does not validate the rule body when it installs the
// package, so this check is the earliest point where these errors are reported.
// Classic templates (engine "", "v1", absent, null, or non-string) are skipped,
// and any other engine string is rejected. package-spec has no structural schema
// for alerting_rule_template files, so classic templates are not content-validated.
// Bytes that are not valid JSON are skipped here; the Kibana object checks
// already report them.

const (
	templateIDMax          = 150
	templateNameMax        = 256
	templateDescriptionMax = 1024
	templateTagMax         = 128
	templateTagsMax        = 20
	templateQueryMax       = 10000
	templateFieldMax       = 256
	templateGroupingMax    = 16
	templatePhaseCountMax  = 1000
	templateArtifactsMax   = 100
	templateRunbookMax     = 50000
	templateDashboardIDMax = 1024
	templateBuilderTypeMax = 64
	templateMinScheduleMS  = int64(5 * 1000)
	templateMaxDurationMS  = int64(365 * 24 * 60 * 60 * 1000)
)

// Durations use Kibana's syntax, copied from DURATION_RE and DURATION_UNIT_TO_MS in
// https://github.com/elastic/kibana/blob/600a8067f05e21388a63ffcc22cf602a3ffb1aa3/x-pack/platform/packages/shared/response-ops/alerting-v2-schemas/src/validation.ts
// This is not Go's time.ParseDuration syntax. Kibana accepts d and w, and it
// rejects decimals and combined values such as 1h30m.
var (
	templateDurationPattern = regexp.MustCompile(`^([0-9]+)(ms|s|m|h|d|w)$`)
	templateDurationUnitMS  = map[string]int64{
		"ms": 1,
		"s":  1000,
		"m":  60 * 1000,
		"h":  60 * 60 * 1000,
		"d":  24 * 60 * 60 * 1000,
		"w":  7 * 24 * 60 * 60 * 1000,
	}
)

type templateProblems []string

func (p *templateProblems) add(format string, args ...any) {
	*p = append(*p, fmt.Sprintf(format, args...))
}

func (p templateProblems) err() error {
	if len(p) == 0 {
		return nil
	}
	return fmt.Errorf("invalid v2 alert rule template: %s", strings.Join(p, "; "))
}

// ValidateAlertRuleTemplates validates Alerting v2 template structure in all
// package validation modes, including built package archives.
func ValidateAlertRuleTemplates(root fspath.FS) specerrors.ValidationErrors {
	files, err := fs.Glob(root, "kibana/alerting_rule_template/*.json")
	if err != nil {
		return specerrors.ValidationErrors{specerrors.NewStructuredErrorf("failed to list alert rule templates: %w", err)}
	}
	var errs specerrors.ValidationErrors
	for _, file := range files {
		data, err := fs.ReadFile(root, file)
		if err != nil {
			errs = append(errs, specerrors.NewStructuredErrorf("failed to read alert rule template %s: %w", root.Path(file), err))
			continue
		}
		if err := validateAlertRuleTemplate(data); err != nil {
			errs = append(errs, specerrors.NewStructuredError(
				fmt.Errorf("file \"%s\" is invalid: %w", root.Path(file), err),
				specerrors.CodeAlertRuleTemplateValidation,
			))
		}
	}
	return errs
}

func validateAlertRuleTemplate(data []byte) error {
	// Read attributes.engine first so classic templates (engine "", "v1",
	// absent, or null, and any non-string engine) are skipped before the full
	// document is type-checked. Otherwise a classic template with a non-string
	// id, type, or engine would fail decoding and be reported under this rule.
	var envelope struct {
		Attributes json.RawMessage `json:"attributes"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		// Not valid JSON. The Kibana object checks already report the parse
		// failure, so tagging it SVR00014 would also flag classic templates.
		return nil
	}
	engine, ok := alertRuleEngineString(envelope.Attributes)
	if !ok || engine == "" || engine == "v1" {
		return nil
	}
	if engine != "v2" {
		return fmt.Errorf("unsupported alert template engine %q", engine)
	}

	var doc struct {
		ID         json.RawMessage `json:"id"`
		Type       json.RawMessage `json:"type"`
		Attributes struct {
			Rule json.RawMessage `json:"rule"`
		} `json:"attributes"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		return err
	}

	var problems templateProblems
	id := jsonString(doc.ID)
	if strings.TrimSpace(id) == "" || templateStringLength(id) > templateIDMax {
		problems.add("id is required and must be at most %d characters", templateIDMax)
	}
	if jsonString(doc.Type) != "alerting_rule_template" {
		problems.add("type must be alerting_rule_template")
	}
	if jsonKind(doc.Attributes.Rule) != "object" {
		problems.add("rule must be an object")
		return problems.err()
	}

	var rule map[string]json.RawMessage
	if err := json.Unmarshal(doc.Attributes.Rule, &rule); err != nil {
		return fmt.Errorf("invalid v2 alert rule template: rule must be an object")
	}
	kind := jsonString(rule["kind"])
	if kind != "alert" && kind != "signal" {
		problems.add("rule.kind must be alert or signal")
	}
	validateTemplateMetadata(kind, rule["metadata"], &problems)
	validateTemplateSchedule(rule["schedule"], &problems)
	validateTemplateTimeField(rule["time_field"], &problems)
	hasBreach := validateTemplateQuery(rule["query"], &problems)
	validateTemplateLifecycle(kind, hasBreach, rule["recovery"], rule["no_data"], rule["state_transition"], &problems)
	validateTemplateGrouping(rule["grouping"], &problems)
	validateTemplateArtifacts(rule["artifacts"], &problems)
	return problems.err()
}

// alertRuleEngineString extracts attributes.engine as a string. The boolean is
// false when attributes is not an object, or when engine is missing, null, or
// not a string. Those cases are treated as classic (skipped) rather than
// reported, so only an explicit string engine reaches the v2/unsupported logic.
func alertRuleEngineString(attributes json.RawMessage) (string, bool) {
	var attrs struct {
		Engine json.RawMessage `json:"engine"`
	}
	if err := json.Unmarshal(attributes, &attrs); err != nil {
		return "", false
	}
	raw := bytes.TrimSpace(attrs.Engine)
	if len(raw) == 0 || string(raw) == "null" {
		return "", false
	}
	var engine string
	if err := json.Unmarshal(raw, &engine); err != nil {
		return "", false
	}
	return engine, true
}

func validateTemplateMetadata(kind string, raw json.RawMessage, problems *templateProblems) {
	obj, ok := jsonObject(raw)
	if !ok {
		problems.add("rule.metadata is required")
		return
	}
	name := jsonString(obj["name"])
	if strings.TrimSpace(name) == "" || templateStringLength(name) > templateNameMax {
		problems.add("rule.metadata.name is required and must be at most %d characters", templateNameMax)
	}
	if raw, exists := obj["description"]; exists {
		description := jsonString(raw)
		if jsonKind(raw) != "string" || strings.TrimSpace(description) == "" || templateStringLength(description) > templateDescriptionMax {
			problems.add("rule.metadata.description must be a non-empty string of at most %d characters", templateDescriptionMax)
		}
	}
	if raw, exists := obj["tags"]; exists {
		validateTemplateTags("rule.metadata.tags", raw, problems)
	}
	if raw, exists := obj["routing_tags"]; exists {
		if kind == "signal" {
			problems.add("signal rules cannot set rule.metadata.routing_tags")
		} else {
			validateTemplateTags("rule.metadata.routing_tags", raw, problems)
		}
	}
	if raw, exists := obj["builder"]; exists {
		builder, ok := jsonObject(raw)
		builderType := jsonString(builder["type"])
		if !ok || strings.TrimSpace(builderType) == "" || templateStringLength(builderType) > templateBuilderTypeMax {
			problems.add("rule.metadata.builder must be an object with a non-empty type of at most %d characters", templateBuilderTypeMax)
		}
	}
}

func validateTemplateTags(field string, raw json.RawMessage, problems *templateProblems) {
	var tags []string
	if err := json.Unmarshal(raw, &tags); jsonKind(raw) != "array" || err != nil {
		problems.add("%s must be an array of strings", field)
		return
	}
	if len(tags) == 0 || len(tags) > templateTagsMax {
		problems.add("%s must contain 1 to %d items", field, templateTagsMax)
	}
	for _, tag := range tags {
		if strings.TrimSpace(tag) == "" || templateStringLength(tag) > templateTagMax {
			problems.add("%s entries must be non-empty strings of at most %d characters", field, templateTagMax)
			return
		}
	}
}

func validateTemplateSchedule(raw json.RawMessage, problems *templateProblems) {
	obj, ok := jsonObject(raw)
	if !ok {
		problems.add("rule.schedule is required")
		return
	}
	every := jsonString(obj["every"])
	if ms, err := parseTemplateDuration(every); err != nil || ms < templateMinScheduleMS || ms > templateMaxDurationMS {
		problems.add("rule.schedule.every must be a duration from 5s to 365d")
	}
	if lookback, exists := obj["lookback"]; exists {
		if ms, err := parseTemplateDuration(jsonString(lookback)); err != nil || ms > templateMaxDurationMS {
			problems.add("rule.schedule.lookback must be a duration up to 365d")
		}
	}
}

func validateTemplateTimeField(raw json.RawMessage, problems *templateProblems) {
	if !jsonPresent(raw) {
		return
	}
	value := jsonString(raw)
	if jsonKind(raw) != "string" || strings.TrimSpace(value) == "" || templateStringLength(value) > templateFieldMax {
		problems.add("rule.time_field must be a non-empty string of at most %d characters", templateFieldMax)
	}
}

func validateTemplateQuery(raw json.RawMessage, problems *templateProblems) bool {
	obj, ok := jsonObject(raw)
	if !ok {
		problems.add("rule.query is required")
		return false
	}
	if _, exists := obj["format"]; exists {
		problems.add("rule.query.format is not part of Alerting v2; use query.base and optional query.breach.segment. Package validation does not parse ES|QL")
	}
	base := jsonString(obj["base"])
	if jsonKind(obj["base"]) != "string" || strings.TrimSpace(base) == "" {
		problems.add("rule.query.base is required")
	} else if templateStringLength(base) > templateQueryMax {
		problems.add("rule.query.base exceeds %d characters", templateQueryMax)
	}
	breach, exists := obj["breach"]
	if !exists {
		return false
	}
	breachObj, ok := jsonObject(breach)
	if !ok {
		problems.add("rule.query.breach must be an object")
		return false
	}
	if _, exists := breachObj["query"]; exists {
		problems.add("rule.query.breach uses query; Alerting v2 breach parts use segment. Package validation does not parse ES|QL")
	}
	segment := jsonString(breachObj["segment"])
	if jsonKind(breachObj["segment"]) != "string" || strings.TrimSpace(segment) == "" {
		problems.add("rule.query.breach.segment is required")
	} else if templateStringLength(segment) > templateQueryMax {
		problems.add("rule.query.breach.segment exceeds %d characters", templateQueryMax)
	}
	return true
}

func validateTemplateLifecycle(kind string, hasBreach bool, recoveryRaw, noDataRaw, transitionRaw json.RawMessage, problems *templateProblems) {
	recoverySet := jsonPresent(recoveryRaw)
	noDataSet := jsonPresent(noDataRaw)
	if kind == "signal" {
		if recoverySet || noDataSet {
			problems.add("signal rules cannot set recovery or no_data")
		}
		if jsonPresent(transitionRaw) {
			problems.add("signal rules cannot set state_transition")
		}
		return
	}
	if kind != "alert" {
		return
	}
	recovery, recoveryOK := jsonObject(recoveryRaw)
	if !recoveryOK {
		problems.add("alert rules require recovery")
	}
	noData, noDataOK := jsonObject(noDataRaw)
	if !noDataOK {
		problems.add("alert rules require no_data")
	}
	recoveringSet := phasePresent(transitionRaw, "recovering")
	if recoveryOK {
		strategy := jsonString(recovery["strategy"])
		switch strategy {
		case "no_breach", "manual":
			if jsonPresent(recovery["segment"]) || jsonPresent(recovery["query"]) {
				problems.add("recovery.strategy %s does not take segment or query", strategy)
			}
			if strategy == "manual" && recoveringSet {
				problems.add("manual recovery cannot set state_transition.recovering")
			}
		case "condition":
			if !hasBreach {
				problems.add("recovery condition requires query.breach")
			}
			segment := jsonString(recovery["segment"])
			if jsonKind(recovery["segment"]) != "string" || strings.TrimSpace(segment) == "" {
				problems.add("recovery condition requires segment")
			}
			if jsonPresent(recovery["query"]) {
				problems.add("recovery condition does not take query")
			}
		case "query":
			query := jsonString(recovery["query"])
			if jsonKind(recovery["query"]) != "string" || strings.TrimSpace(query) == "" {
				problems.add("recovery query strategy requires query")
			}
			if jsonPresent(recovery["segment"]) {
				problems.add("recovery query strategy does not take segment")
			}
		default:
			problems.add("recovery.strategy must be no_breach, condition, query, or manual")
		}
	}
	if noDataOK {
		strategy := jsonString(noData["strategy"])
		switch strategy {
		case "ignore":
			if jsonPresent(noData["query"]) {
				problems.add("no_data ignore does not take query")
			}
		case "keep_last", "resolve":
			if raw, exists := noData["query"]; exists {
				query := jsonString(raw)
				if jsonKind(raw) != "string" || strings.TrimSpace(query) == "" || templateStringLength(query) > templateQueryMax {
					problems.add("no_data.query must be a non-empty string of at most %d characters", templateQueryMax)
				}
			}
			if !hasBreach && !jsonPresent(noData["query"]) {
				problems.add("no_data %s requires query.breach or no_data.query", strategy)
			}
		default:
			problems.add("no_data.strategy must be ignore, keep_last, or resolve")
		}
	}
	validateTemplateTransition(transitionRaw, problems)
}

func validateTemplateTransition(raw json.RawMessage, problems *templateProblems) {
	if !jsonPresent(raw) {
		return
	}
	obj, ok := jsonObject(raw)
	if !ok {
		problems.add("state_transition must be an object")
		return
	}
	for _, phase := range []string{"pending", "recovering"} {
		phaseRaw, exists := obj[phase]
		if !exists {
			continue
		}
		phaseObj, ok := jsonObject(phaseRaw)
		if !ok {
			problems.add("state_transition.%s must be an object", phase)
			continue
		}
		_, hasCount := phaseObj["count"]
		_, hasTimeframe := phaseObj["timeframe"]
		if !hasCount && !hasTimeframe {
			problems.add("state_transition.%s must set count or timeframe", phase)
		}
		if hasCount {
			var count float64
			if err := json.Unmarshal(phaseObj["count"], &count); jsonKind(phaseObj["count"]) == "null" || err != nil || count < 0 || count > templatePhaseCountMax || math.Trunc(count) != count {
				problems.add("state_transition.%s.count must be an integer from 0 to %d", phase, templatePhaseCountMax)
			}
		}
		if hasTimeframe {
			if ms, err := parseTemplateDuration(jsonString(phaseObj["timeframe"])); err != nil || ms > templateMaxDurationMS {
				problems.add("state_transition.%s.timeframe must be a duration up to 365d", phase)
			}
		}
		operator, hasOperator := phaseObj["operator"]
		if !hasOperator {
			continue
		}
		op := jsonString(operator)
		if (op != "and" && op != "or") || !hasCount || !hasTimeframe {
			problems.add("state_transition.%s.operator must be and or or and requires count and timeframe", phase)
		}
	}
}

func phasePresent(transition json.RawMessage, phase string) bool {
	obj, ok := jsonObject(transition)
	if !ok {
		return false
	}
	raw, exists := obj[phase]
	return exists && jsonKind(raw) != "null"
}

func validateTemplateGrouping(raw json.RawMessage, problems *templateProblems) {
	if !jsonPresent(raw) {
		return
	}
	obj, ok := jsonObject(raw)
	if !ok {
		problems.add("rule.grouping must be an object")
		return
	}
	fieldsRaw, exists := obj["fields"]
	if !exists || jsonKind(fieldsRaw) == "null" {
		problems.add("rule.grouping.fields must be an array")
		return
	}
	if jsonKind(fieldsRaw) != "array" {
		problems.add("rule.grouping.fields must be an array")
		return
	}
	var fields []string
	if err := json.Unmarshal(fieldsRaw, &fields); err != nil {
		problems.add("rule.grouping.fields entries must be strings")
		return
	}
	if len(fields) == 0 || len(fields) > templateGroupingMax {
		problems.add("rule.grouping.fields must contain 1 to %d fields", templateGroupingMax)
	}
	for _, field := range fields {
		if strings.TrimSpace(field) == "" || templateStringLength(field) > templateFieldMax {
			problems.add("rule.grouping.fields contains an empty or oversized field")
			return
		}
	}
}

func validateTemplateArtifacts(raw json.RawMessage, problems *templateProblems) {
	if !jsonPresent(raw) {
		return
	}
	if jsonKind(raw) != "array" {
		problems.add("rule.artifacts must be an array")
		return
	}
	var artifacts []map[string]json.RawMessage
	if err := json.Unmarshal(raw, &artifacts); err != nil {
		problems.add("rule.artifacts entries must be objects")
		return
	}
	if len(artifacts) == 0 || len(artifacts) > templateArtifactsMax {
		problems.add("rule.artifacts must contain 1 to %d items", templateArtifactsMax)
	}
	seen := map[string]struct{}{}
	for _, artifact := range artifacts {
		id := jsonString(artifact["id"])
		kind := jsonString(artifact["type"])
		if strings.TrimSpace(id) == "" || jsonKind(artifact["type"]) != "string" || strings.TrimSpace(kind) == "" {
			problems.add("rule.artifacts entries require id and type")
			return
		}
		if _, exists := seen[id]; exists {
			problems.add("rule.artifacts ids must be unique")
			return
		}
		seen[id] = struct{}{}
		data, ok := jsonObject(artifact["data"])
		if !ok {
			problems.add("rule.artifacts data must be an object")
			return
		}
		switch kind {
		case "runbook":
			content := jsonString(data["content"])
			if jsonKind(data["content"]) != "string" || strings.TrimSpace(content) == "" || templateStringLength(content) > templateRunbookMax {
				problems.add("runbook artifact data.content must be a non-empty string")
			}
		case "dashboard":
			if _, exists := data["dashboardId"]; exists {
				problems.add("dashboard artifact data.dashboardId is not accepted; use dashboard_id")
			}
			dashboardID := jsonString(data["dashboard_id"])
			if jsonKind(data["dashboard_id"]) != "string" || strings.TrimSpace(dashboardID) == "" || templateStringLength(dashboardID) > templateDashboardIDMax {
				problems.add("dashboard artifact data.dashboard_id is required")
			}
		}
	}
}

// templateStringLength matches JavaScript String.length, which Kibana's schemas
// use for string limits. Characters outside the BMP occupy two UTF-16 units.
func templateStringLength(value string) int {
	length := 0
	for _, r := range value {
		length++
		if r > 0xffff {
			length++
		}
	}
	return length
}

func parseTemplateDuration(value string) (int64, error) {
	match := templateDurationPattern.FindStringSubmatch(value)
	if match == nil {
		return 0, fmt.Errorf("%q is not a duration", value)
	}
	n, err := strconv.ParseInt(match[1], 10, 64)
	if err != nil {
		return 0, err
	}
	unit := templateDurationUnitMS[match[2]]
	if unit == 0 || n > math.MaxInt64/unit {
		return 0, fmt.Errorf("%q is not a duration", value)
	}
	return n * unit, nil
}

func jsonKind(raw json.RawMessage) string {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		return "missing"
	}
	switch trimmed[0] {
	case '{':
		return "object"
	case '[':
		return "array"
	case '"':
		return "string"
	case 'n':
		return "null"
	default:
		return "other"
	}
}

// jsonPresent distinguishes omission from a supplied value, including null.
func jsonPresent(raw json.RawMessage) bool {
	return jsonKind(raw) != "missing"
}

func jsonObject(raw json.RawMessage) (map[string]json.RawMessage, bool) {
	if jsonKind(raw) != "object" {
		return nil, false
	}
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(raw, &obj); err != nil {
		return nil, false
	}
	return obj, true
}

func jsonString(raw json.RawMessage) string {
	if jsonKind(raw) != "string" {
		return ""
	}
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return ""
	}
	return value
}
