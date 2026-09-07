# Validation Codes

| Validation          | Short description                     |
|---------------------|---------------------------------------|
| **[JSE][JSE00001]** | **JSON Schema Errors**                |
| [JSE00001]          | Rename message to event.original      |
| **[PSR][PSR00001]** | **Package Spec Rule**                 |
| [PSR00001]          | Non GA spec used in GA package        |
| [PSR00002]          | Prerelease feature used in GA package |
| **[SVR][SVR00001]** | **Semantic Validation Rules**         |
| [SVR00001]          | Dashboard with query but no filter    |
| [SVR00002]          | Dashboard without filter              |
| [SVR00003]          | Dangling object IDs                   |
| [SVR00004]          | Visualization by value                |
| [SVR00005]          | Minimum Kibana version                |
| [SVR00006]          | Processor tag is required             |
| [SVR00007]          | Kibana tag is duplicate               |
| [SVR00008]          | Pipeline failure handler must set event.kind    |
| [SVR00009]          | Pipeline failure handler must set error.message |
| [SVR00011]          | Grok pattern contains text that is not a token  |

## JSE00001 - Rename message to event.original
[JSE00001]: #jse00001---rename-message-to-eventoriginal

**Available since [3.1.0](https://github.com/elastic/package-spec/releases/tag/v3.1.0)**

## PSR00001 - Non GA spec used in GA package
[PSR00001]: #psr00001---non-ga-spec-used-in-ga-package

**Available since [3.0.1](https://github.com/elastic/package-spec/releases/tag/v3.0.1)**

## PSR00002 - Prerelease feature used in GA package
[PSR00002]: #psr00002---prerelease-feature-used-in-ga-package

**Available since [3.0.0](https://github.com/elastic/package-spec/releases/tag/v3.0.0)**

## SVR00001 - Dashboard with query but no filter
[SVR00001]: #svr00001---dashboard-with-query-but-no-filter

**Available since [2.13.0](https://github.com/elastic/package-spec/releases/tag/v2.13.0)**

## SVR00002 - Dashboard without filter
[SVR00002]: #svr00002---dashboard-without-filter

**Available since [2.13.0](https://github.com/elastic/package-spec/releases/tag/v2.13.0)**

## SVR00003 - Dangling object IDs
[SVR00003]: #svr00003---dangling-object-ids

**Available since [2.13.0](https://github.com/elastic/package-spec/releases/tag/v2.13.0)**

## SVR00004 - Visualization by value
[SVR00004]: #svr00004---visualization-by-value

**Available since [3.0.0](https://github.com/elastic/package-spec/releases/tag/v3.0.0)**

## SVR00005 - Minimum Kibana version
[SVR00005]: #svr00005---minimum-kibana-version

**Available since [3.0.0](https://github.com/elastic/package-spec/releases/tag/v3.0.0)**

## SVR00006 - Processor tag is required
[SVR00006]: #svr00006---processor-tag-is-required

**Available since [3.6.0](https://github.com/elastic/package-spec/releases/tag/v3.6.0)**

Every processor in an ingest pipeline must include a unique tag, which is used to
annotate the processor in metrics and logs. Processors in the global pipeline
on_failure handler are excluded from this check.

```yaml
set:
  tag: set_event_category
  field: event.category
  value: [network]
```

## SVR00007 - Kibana tag is duplicate
[SVR00007]: #svr00007---kibana-tag-is-duplicate

**Available since [3.5.5](https://github.com/elastic/package-spec/releases/tag/v3.5.5)**

Kibana tags declared under `kibana/tags.yml` are duplicated or package tags under `kibana/tag` directory are sharing the same id.

## SVR00008 - Pipeline failure handler must set event.kind
[SVR00008]: #svr00008---pipeline-failure-handler-must-set-eventkind

**Available since [3.6.0](https://github.com/elastic/package-spec/releases/tag/v3.6.0)**

The global on_failure handler for an ingest pipeline must set `event.kind` to
`pipeline_error`. This value indicates that an error occurred during the
ingestion of this event, and that event data may be missing, inconsistent,
or incorrect. 

```yaml
on_failure:
  - set:
      field: event.kind
      value: pipeline_error
```

## SVR00009 - Pipeline failure handler must set error.message
[SVR00009]: #svr00009---pipeline-failure-handler-must-set-errormessage

**Available since [3.6.0](https://github.com/elastic/package-spec/releases/tag/v3.6.0)**

The global on_failure handler for an ingest pipeline must set or append to
`error.message` and the value must include the following:

- `_ingest.on_failure_processor_type`
- `_ingest.on_failure_processor_tag`
- `_ingest.on_failure_message`
- `_ingest.pipeline`

In cases where more than one `error.message` value is expected or could occur,
the `append` processor should be used.

```yaml
on_failure:
  - append:
      field: error.message
      value: >-
        Processor '{{{ _ingest.on_failure_processor_type }}}'
        with tag '{{{ _ingest.on_failure_processor_tag }}}'
        in pipeline '{{{ _ingest.pipeline }}}'
        failed with message '{{{ _ingest.on_failure_message }}}'
```

```yaml
on_failure:
  - set:
      field: error.message
      value: >-
        Processor '{{{ _ingest.on_failure_processor_type }}}'
        with tag '{{{ _ingest.on_failure_processor_tag }}}'
        in pipeline '{{{ _ingest.pipeline }}}'
        failed with message '{{{ _ingest.on_failure_message }}}'
```

## SVR00011 - Grok pattern contains text that is not a token
[SVR00011]: #svr00011---grok-pattern-contains-text-that-is-not-a-token

**Available since [3.7.0](https://github.com/elastic/package-spec/releases/tag/v3.7.0)**
(reported as a warning for earlier spec versions)

Every `%{` in a grok processor's `patterns` and `pattern_definitions` must start a
grok token: `%{NAME}`, `%{NAME:field}`, `%{NAME:field:type}` or `%{NAME=definition}`,
where `NAME` is letters, digits and underscores and `field` may also contain `.`, `-`,
`@`, `[`, `]` and `:`.

Grok does not report text that fails to parse as a token. It leaves it in the regular
expression as-is, so the processor compiles and runs, and the capture the author wrote
simply never exists. Because the line then either never matches or matches without the
field, and because a catch-all pattern or `ignore_failure: true` is usually nearby, nothing
downstream reports the problem either.

```yaml
# Reported: `|` cannot appear in a pattern name, so `%{USERNAME` is literal text and the
# `|` becomes an alternation of the whole pattern.
- '^Logout handler : %{DATA}, for user <%{USERNAME|EMAILADDRESS:user.name}>$'

# Reported: the colon between the pattern name and the field is missing.
- '^Primary authentication %{WORD_tmp.outcome}'

# Reported: the token is never closed, `\]` is not a `}`.
- '%{GREEDYDATA:log.msg.error\]'

# Accepted.
- '^Logout handler : %{DATA}, for user <(?:%{EMAILADDRESS:user.name}|%{USERNAME:user.name})>$'
- '^Primary authentication %{WORD:_tmp.outcome}'
- '%{GREEDYDATA:log.msg.error}\]'
```

A pattern that needs to match a literal `%{` in the input can exclude this check in
`validation.yml`.
