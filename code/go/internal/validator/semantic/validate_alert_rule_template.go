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
	"unicode"
	"unicode/utf16"
	"unicode/utf8"

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
// and any other engine string is rejected without an excludable error code.
// package-spec has no structural schema for alerting_rule_template files, so
// classic templates are not content-validated.
// Bytes that are not valid JSON are skipped here; the Kibana object checks
// already report them.

const (
	templateIDMax             = 150
	templateNameMax           = 256
	templateDescriptionMax    = 1024
	templateTagMax            = 128
	templateTagsMax           = 20
	templateQueryMax          = 10000
	templateFieldMax          = 256
	templateGroupingMax       = 16
	templatePhaseCountMax     = 1000
	templateArtifactsMax      = 100
	templateRunbookMax        = 50000
	templateDashboardIDMax    = 1024
	templateBuilderTypeMax    = 64
	templateArtifactTypeMax   = 128
	templateArtifactFieldsMax = 32
	// MAX_ARTIFACT_DATA_BYTES in alerting-v2-constants. Kibana compares it with
	// the length of JSON.stringify(data).
	templateArtifactDataMax = 2 * 65536
	templateDurationLenMax  = 32
	templateMinScheduleMS   = int64(5 * 1000)
	templateMaxDurationMS   = int64(365 * 24 * 60 * 60 * 1000)
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

const alertRuleTemplateFiles = "kibana/alerting_rule_template/*.json"

// alertingV2MinFormatVersion is the first package format_version supported by
// Kibana 9.6.0, the first Kibana that installs Alerting v2 templates.
const alertingV2MinFormatVersion = "3.7.0"

type alertRuleTemplate struct {
	ID         json.RawMessage `json:"id"`
	Type       json.RawMessage `json:"type"`
	Attributes json.RawMessage `json:"attributes"`
}

// ValidateAlertRuleTemplatesV2FormatVersion rejects Alerting v2 templates, and
// templates with an unsupported engine, in packages whose format_version is
// older than alertingV2MinFormatVersion. Kibana before 9.6.0 fails to install
// them, and the registry offers those packages only to Kibana versions that
// support their format_version. The errors have no code so validation.yml
// cannot exclude them.
func ValidateAlertRuleTemplatesV2FormatVersion(root fspath.FS) specerrors.ValidationErrors {
	files, err := fs.Glob(root, alertRuleTemplateFiles)
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
		_, skip, err := parseAlertRuleTemplate(data)
		if err != nil {
			errs = append(errs, specerrors.NewStructuredErrorf("file \"%s\" is invalid: %w", root.Path(file), err))
			continue
		}
		if skip {
			continue
		}
		errs = append(errs, specerrors.NewStructuredErrorf(
			"file \"%s\" is invalid: Alerting v2 rule templates (engine v2) require format_version %s or later",
			root.Path(file), alertingV2MinFormatVersion))
	}
	return errs
}

// ValidateAlertRuleTemplates validates Alerting v2 template structure in all
// package validation modes, including built package archives.
func ValidateAlertRuleTemplates(root fspath.FS) specerrors.ValidationErrors {
	files, err := fs.Glob(root, alertRuleTemplateFiles)
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
		template, skip, err := parseAlertRuleTemplate(data)
		if err != nil {
			errs = append(errs, specerrors.NewStructuredErrorf("file \"%s\" is invalid: %w", root.Path(file), err))
			continue
		}
		if skip {
			continue
		}
		if err := validateAlertRuleTemplate(template); err != nil {
			errs = append(errs, specerrors.NewStructuredError(
				fmt.Errorf("file \"%s\" is invalid: %w", root.Path(file), err),
				specerrors.CodeAlertRuleTemplateValidation,
			))
		}
	}
	return errs
}

// parseAlertRuleTemplate separates engine errors from v2 structural findings.
// skip is true for classic templates and invalid JSON. The existing Kibana
// object checks report invalid JSON. RawMessage fields preserve the classic
// skip even when id, type, attributes, or engine have unexpected types.
func parseAlertRuleTemplate(data []byte) (alertRuleTemplate, bool, error) {
	var template alertRuleTemplate
	if err := json.Unmarshal(data, &template); err != nil {
		return template, true, nil
	}
	engine, ok := alertRuleEngineString(template.Attributes)
	if !ok || engine == "" || engine == "v1" {
		return template, true, nil
	}
	if engine != "v2" {
		return template, false, fmt.Errorf("unsupported alert template engine %q", engine)
	}
	return template, false, nil
}

func validateAlertRuleTemplate(template alertRuleTemplate) error {
	attributes, _ := jsonObject(template.Attributes)

	var problems templateProblems
	id := jsonString(template.ID)
	if id == "" || templateStringLength(id) > templateIDMax {
		problems.add("id is required and must be at most %d characters", templateIDMax)
	}
	if jsonString(template.Type) != "alerting_rule_template" {
		problems.add("type must be alerting_rule_template")
	}
	rule, ok := jsonObject(attributes["rule"])
	if !ok {
		problems.add("rule must be an object")
		return problems.err()
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
	if name == "" || templateStringLength(name) > templateNameMax {
		problems.add("rule.metadata.name is required and must be at most %d characters", templateNameMax)
	}
	if raw, exists := obj["description"]; exists {
		description := jsonString(raw)
		if jsonKind(raw) != "string" || isBlank(description) || templateStringLength(description) > templateDescriptionMax {
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
		if !ok || isBlank(builderType) || templateStringLength(builderType) > templateBuilderTypeMax {
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
		if tag == "" || templateStringLength(tag) > templateTagMax {
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
	if jsonKind(raw) != "string" || value == "" || templateStringLength(value) > templateFieldMax {
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
	if jsonKind(obj["base"]) != "string" || isBlank(base) {
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
	if jsonKind(breachObj["segment"]) != "string" || isBlank(segment) {
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
			if jsonKind(recovery["segment"]) != "string" || isBlank(segment) {
				problems.add("recovery condition requires segment")
			} else if templateStringLength(segment) > templateQueryMax {
				problems.add("recovery.segment exceeds %d characters", templateQueryMax)
			}
			if jsonPresent(recovery["query"]) {
				problems.add("recovery condition does not take query")
			}
		case "query":
			query := jsonString(recovery["query"])
			if jsonKind(recovery["query"]) != "string" || isBlank(query) {
				problems.add("recovery query strategy requires query")
			} else if templateStringLength(query) > templateQueryMax {
				problems.add("recovery.query exceeds %d characters", templateQueryMax)
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
				if jsonKind(raw) != "string" || isBlank(query) || templateStringLength(query) > templateQueryMax {
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
		if field == "" || templateStringLength(field) > templateFieldMax {
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
	for i, artifact := range artifacts {
		field := fmt.Sprintf("rule.artifacts[%d]", i)
		id := jsonString(artifact["id"])
		kind := jsonString(artifact["type"])
		if id == "" || templateStringLength(id) > templateIDMax {
			problems.add("%s.id is required and must be at most %d characters", field, templateIDMax)
		} else if _, exists := seen[id]; exists {
			problems.add("rule.artifacts ids must be unique")
		}
		seen[id] = struct{}{}
		if kind == "" || templateStringLength(kind) > templateArtifactTypeMax {
			problems.add("%s.type is required and must be at most %d characters", field, templateArtifactTypeMax)
		}
		data, ok := jsonObject(artifact["data"])
		if !ok {
			problems.add("%s.data must be an object", field)
			continue
		}
		validateTemplateArtifactData(field, kind, artifact["data"], data, problems)
	}
}

func validateTemplateArtifactData(field, kind string, raw json.RawMessage, data map[string]json.RawMessage, problems *templateProblems) {
	if len(data) > templateArtifactFieldsMax {
		problems.add("%s.data must have at most %d fields", field, templateArtifactFieldsMax)
	}
	for key := range data {
		if key == "" || templateStringLength(key) > templateFieldMax {
			problems.add("%s.data keys must be 1 to %d characters", field, templateFieldMax)
			break
		}
	}
	if templateJSONLength(raw) > templateArtifactDataMax {
		problems.add("%s.data must not exceed %d characters when serialized", field, templateArtifactDataMax)
	}
	switch kind {
	case "runbook":
		content := jsonString(data["content"])
		if jsonKind(data["content"]) != "string" || isBlank(content) || templateStringLength(content) > templateRunbookMax {
			problems.add("runbook artifact data.content must be a non-empty string")
		}
	case "dashboard":
		if _, exists := data["dashboardId"]; exists {
			problems.add("dashboard artifact data.dashboardId is not accepted; use dashboard_id")
		}
		dashboardID := jsonString(data["dashboard_id"])
		if jsonKind(data["dashboard_id"]) != "string" || isBlank(dashboardID) || templateStringLength(dashboardID) > templateDashboardIDMax {
			problems.add("dashboard artifact data.dashboard_id is required")
		}
	}
}

// isBlank matches Kibana's s.trim() === "" checks. JavaScript whitespace is
// Unicode White_Space without U+0085, plus U+FEFF.
func isBlank(value string) bool {
	return strings.TrimFunc(value, func(r rune) bool {
		return r == '\uFEFF' || (r != '\u0085' && unicode.IsSpace(r))
	}) == ""
}

// templateJSONLength returns JSON.stringify(JSON.parse(raw)).length, the value
// Kibana compares with MAX_ARTIFACT_DATA_BYTES. raw must be valid JSON.
func templateJSONLength(raw json.RawMessage) int {
	switch jsonKind(raw) {
	case "object":
		var obj map[string]json.RawMessage
		if err := json.Unmarshal(raw, &obj); err != nil {
			return 0
		}
		length := 2 + max(len(obj)-1, 0)
		for key, value := range obj {
			// encoding/json decodes keys, so a lone surrogate in a key counts as U+FFFD.
			length += jsStringifiedLength(utf16.Encode([]rune(key))) + 1 + templateJSONLength(value)
		}
		return length
	case "array":
		var items []json.RawMessage
		if err := json.Unmarshal(raw, &items); err != nil {
			return 0
		}
		length := 2 + max(len(items)-1, 0)
		for _, item := range items {
			length += templateJSONLength(item)
		}
		return length
	case "string":
		return jsStringifiedLength(jsonStringUnits(raw))
	}
	literal := string(bytes.TrimSpace(raw))
	switch literal {
	case "true", "false", "null":
		return len(literal)
	}
	number, _ := strconv.ParseFloat(literal, 64)
	if math.IsInf(number, 0) {
		return len("null")
	}
	return len(jsNumberString(number))
}

// jsonStringUnits decodes a JSON string literal into UTF-16 code units,
// keeping lone surrogates that encoding/json would replace with U+FFFD.
func jsonStringUnits(raw json.RawMessage) []uint16 {
	literal := string(bytes.TrimSpace(raw))
	literal = literal[1 : len(literal)-1]
	escapes := map[byte]uint16{'"': '"', '\\': '\\', '/': '/', 'b': '\b', 'f': '\f', 'n': '\n', 'r': '\r', 't': '\t'}
	var units []uint16
	for i := 0; i < len(literal); {
		if literal[i] != '\\' {
			r, size := utf8.DecodeRuneInString(literal[i:])
			units = utf16.AppendRune(units, r)
			i += size
			continue
		}
		if literal[i+1] == 'u' {
			unit, _ := strconv.ParseUint(literal[i+2:i+6], 16, 16)
			units = append(units, uint16(unit))
			i += 6
			continue
		}
		units = append(units, escapes[literal[i+1]])
		i += 2
	}
	return units
}

// jsStringifiedLength returns the length of JSON.stringify for a string of
// UTF-16 code units, including the quotes.
func jsStringifiedLength(units []uint16) int {
	length := 2
	for i := 0; i < len(units); i++ {
		unit := units[i]
		switch {
		case unit >= 0xD800 && unit <= 0xDBFF && i+1 < len(units) && units[i+1] >= 0xDC00 && units[i+1] <= 0xDFFF:
			length += 2
			i++
		case unit >= 0xD800 && unit <= 0xDFFF:
			length += len(`\ud800`)
		case unit == '"' || unit == '\\' || unit == '\b' || unit == '\f' || unit == '\n' || unit == '\r' || unit == '\t':
			length += 2
		case unit < 0x20:
			length += len(`\u0000`)
		default:
			length++
		}
	}
	return length
}

// jsNumberString formats a number like JavaScript's Number.prototype.toString.
func jsNumberString(number float64) string {
	if number == 0 {
		return "0"
	}
	sign := ""
	if number < 0 {
		sign, number = "-", -number
	}
	mantissa, exponent, _ := strings.Cut(strconv.FormatFloat(number, 'e', -1, 64), "e")
	digits := strings.Replace(mantissa, ".", "", 1)
	power, _ := strconv.Atoi(exponent)
	k, n := len(digits), power+1
	switch {
	case k <= n && n <= 21:
		return sign + digits + strings.Repeat("0", n-k)
	case 0 < n && n <= 21:
		return sign + digits[:n] + "." + digits[n:]
	case -6 < n && n <= 0:
		return sign + "0." + strings.Repeat("0", -n) + digits
	}
	exp := "e+" + strconv.Itoa(n-1)
	if n-1 < 0 {
		exp = "e-" + strconv.Itoa(1-n)
	}
	if k == 1 {
		return sign + digits + exp
	}
	return sign + digits[:1] + "." + digits[1:] + exp
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
	if len(value) > templateDurationLenMax {
		return 0, fmt.Errorf("%q is longer than %d characters", value, templateDurationLenMax)
	}
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
