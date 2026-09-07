// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package semantic

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestValidatePipelineGrok(t *testing.T) {
	testCases := []struct {
		name     string
		pipeline string
		errors   []string
	}{
		{
			name: "good",
			pipeline: `
processors:
  - grok:
      tag: grok_message
      field: message
      patterns:
        - '%{IP:source.ip} %{WORD:http.request.method} %{NUMBER:http.response.status_code:int}'
        - '%{TIMESTAMP_ISO8601:_tmp.ts} %{GREEDYDATA:message}'
        - '(?<custom.raw_group>[a-z]+) %{DATA}'
        - '%{SESSION}(?: %{WORD:_tmp.outcome})?$'
        - '%{USER:user.name}@%{HOSTNAME:user.domain} %{NAME=[a-z]+}'
      pattern_definitions:
        SESSION: '\(session:%{SPACE}%{NOTSPACE:session.id}\)'
        NAME: '[a-z_]+'
        UNSIGNED_INT: '[0-9]+'
      on_failure:
        - grok:
            tag: grok_fallback
            field: message
            patterns:
              - '%{GREEDYDATA:event.original}'
  - foreach:
      tag: foreach_values
      field: values
      processor:
        grok:
          field: _ingest._value
          patterns:
            - '%{IP:related.ip}'
on_failure:
  - grok:
      field: error.message
      patterns:
        - '%{GREEDYDATA:error.detail}'
`,
		},
		{
			name: "alternation inside a pattern name",
			pipeline: `
processors:
  - grok:
      tag: grok_message
      field: message
      patterns:
        - '^Logout handler : %{DATA}, for user <%{USERNAME|EMAILADDRESS:user.name}>$'
`,
			errors: []string{
				`file "default.yml" is invalid: grok processor at line 3 has "%{USERNAME|EMAILADDRESS:user.name}" in patterns[0], which is not a grok token and is matched as literal text (SVR00011)`,
			},
		},
		{
			name: "character class where a pattern name goes",
			pipeline: `
processors:
  - grok:
      tag: grok_id
      field: pslogid
      patterns:
        - '%{UUID:request.id}'
        - '%{[A-Fa-f0-9]{32}:request.id}'
`,
			errors: []string{
				`file "default.yml" is invalid: grok processor at line 3 has "%{[A-Fa-f0-9]{32}" in patterns[1], which is not a grok token and is matched as literal text (SVR00011)`,
			},
		},
		{
			name: "missing closing brace",
			pipeline: `
processors:
  - grok:
      tag: grok_message
      field: message
      patterns:
        - '%{WORD:network.direction} \[%{WORD:log.msg.type}\s+%{GREEDYDATA:log.msg.error\]'
`,
			errors: []string{
				`file "default.yml" is invalid: grok processor at line 3 has "%{GREEDYDATA:log.msg.error\\]" in patterns[0], which is not a grok token and is matched as literal text (SVR00011)`,
			},
		},
		{
			name: "missing colon before the field name",
			pipeline: `
processors:
  - set:
      tag: set_kind
      field: event.kind
      value: event
  - grok:
      tag: grok_message
      field: message
      patterns:
        - 'Login %{WORD:_tmp.outcome}'
        - '^Primary authentication %{WORD_tmp.outcome}'
`,
			errors: []string{
				`file "default.yml" is invalid: grok processor at line 7 has "%{WORD_tmp.outcome}" in patterns[1], which is not a grok token and is matched as literal text (SVR00011)`,
			},
		},
		{
			name: "wrong closing character in a pattern definition",
			pipeline: `
processors:
  - grok:
      tag: grok_message
      field: message
      patterns:
        - '%{ECS_SYSLOG_PRI}%{GREEDYDATA:message}'
      pattern_definitions:
        ECS_SYSLOG_PRI: '<%{NONNEGINT:log.syslog.priority>'
`,
			errors: []string{
				`file "default.yml" is invalid: grok processor at line 3 has "%{NONNEGINT:log.syslog.priority>" in pattern_definitions["ECS_SYSLOG_PRI"], which is not a grok token and is matched as literal text (SVR00011)`,
			},
		},
		{
			name: "several in one pattern are each reported",
			pipeline: `
processors:
  - grok:
      tag: grok_message
      field: message
      patterns:
        - '%{IP:source.ip} %{WORD source.port} %{WORD:ok} %{NUMBER}%{ bad}'
`,
			errors: []string{
				`file "default.yml" is invalid: grok processor at line 3 has "%{WORD source.port}" in patterns[0], which is not a grok token and is matched as literal text (SVR00011)`,
				`file "default.yml" is invalid: grok processor at line 3 has "%{ bad}" in patterns[0], which is not a grok token and is matched as literal text (SVR00011)`,
			},
		},
		{
			name: "inside a foreach processor",
			pipeline: `
processors:
  - foreach:
      tag: foreach_values
      field: values
      processor:
        grok:
          field: _ingest._value
          patterns:
            - '%{IP related.ip}'
`,
			errors: []string{
				`file "default.yml" is invalid: grok processor at line 3 has "%{IP related.ip}" in patterns[0], which is not a grok token and is matched as literal text (SVR00011)`,
			},
		},
		{
			name: "inside a processor on_failure handler",
			pipeline: `
processors:
  - grok:
      tag: grok_message
      field: message
      patterns:
        - '%{GREEDYDATA:message}'
      on_failure:
        - grok:
            tag: grok_fallback
            field: message
            patterns:
              - '%{GREEDYDATA message}'
`,
			errors: []string{
				`file "default.yml" is invalid: grok processor at line 9 has "%{GREEDYDATA message}" in patterns[0], which is not a grok token and is matched as literal text (SVR00011)`,
			},
		},
		{
			name: "inside the pipeline on_failure handler",
			pipeline: `
processors:
  - set:
      tag: set_kind
      field: event.kind
      value: event
on_failure:
  - grok:
      field: error.message
      patterns:
        - '%{GREEDYDATA error.detail}'
`,
			errors: []string{
				`file "default.yml" is invalid: grok processor at line 8 has "%{GREEDYDATA error.detail}" in patterns[0], which is not a grok token and is matched as literal text (SVR00011)`,
			},
		},
		{
			name: "unterminated text is excerpted rather than dumped",
			pipeline: `
processors:
  - grok:
      tag: grok_message
      field: message
      patterns:
        - '%{GREEDYDATA:a.very.long.field.name.that.keeps.going.without.ever.closing.the.token.at.all'
`,
			errors: []string{
				`file "default.yml" is invalid: grok processor at line 3 has "%{GREEDYDATA:a.very.long.field.name.that.keeps.going.without..." in patterns[0], which is not a grok token and is matched as literal text (SVR00011)`,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var pipeline ingestPipeline
			err := yaml.Unmarshal([]byte(tc.pipeline), &pipeline)
			require.NoError(t, err)

			errors := validatePipelineGrok(&pipeline, "default.yml")
			var messages []string
			for _, err := range errors {
				messages = append(messages, err.Error())
			}
			assert.ElementsMatch(t, tc.errors, messages)
		})
	}
}

func TestGrokTokenMatchesWhatGrokAccepts(t *testing.T) {
	// The pattern name class is Grok.java's `[A-z0-9]`, an ASCII range that also admits
	// `[`, `\`, `]`, `^`, `_` and the backtick. These are odd, but they are tokens to
	// Elasticsearch, so they must be tokens here or we would report spurious errors.
	for _, token := range []string{
		"%{WORD}",
		"%{WORD:field}",
		"%{WORD:field:int}",
		"%{WORD:some.nested-field_name@[0]}",
		"%{NAME=[a-z]+}",
		"%{NAME:field=\\d+\\.\\d+}",
		"%{_UNDERSCORE_}",
		"%{[BRACKETS]}",
	} {
		assert.True(t, grokToken.MatchString(token), token)
	}
	for _, text := range []string{
		"%{}",
		"%{ WORD}",
		"%{WORD field}",
		"%{WORD|OTHER:field}",
		"%{WORD:field",
		"%{WORD:field>",
		"%{WORD-DASH}",
		"%{WORD:field\\]}",
	} {
		assert.False(t, grokToken.MatchString(text), text)
	}
}
