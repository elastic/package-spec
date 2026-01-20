# OTel Multi-Signal Collector

The OTel Multi-Signal Collector integration allows you to collect logs, metrics, and traces using a single OpenTelemetry Collector receiver.

Use the OTel Multi-Signal Collector integration to collect multiple signal types from a single receiver. This demonstrates the `available_types` feature that allows OTel input packages to declare multiple signal types.

## Data streams

The OTel Multi-Signal Collector integration can collect three types of data streams: logs, metrics, and traces.

**Logs** help you keep a record of events happening in your system.

**Metrics** give you insight into the state of your system.

**Traces** provide distributed tracing information.

## Requirements

You need Elasticsearch for storing and searching your data and Kibana for visualizing and managing it.
You can use our hosted Elasticsearch Service on Elastic Cloud, which is recommended, or self-manage the Elastic Stack on your own hardware.

## Setup

For step-by-step instructions on how to set up an integration, see the
[Getting started](https://www.elastic.co/guide/en/welcome-to-elastic/current/getting-started-observability.html) guide.
