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
| [SVR00007]          | Processor tag duplicated in pipeline  |

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
annotate the processor in metrics and logs.

```yaml
set:
  tag: set_event_category
  field: event.category
  value: [network]
```

## SVR00007 - Processor tag duplicated in pipeline
[SVR00007]: #svr00007---processor-tag-duplicated-in-pipeline

**Available since [3.6.0](https://github.com/elastic/package-spec/releases/tag/v3.6.0)**

A processor tag must not be repeated in an ingest pipeline. A tag must uniquely
identify a processor in a pipeline so it can be annotated in metrics and logs.
