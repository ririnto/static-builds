# Vector Static Build

This target packages the upstream `x86_64-unknown-linux-musl` Vector
binary as a static artifact.

## Modules and Features

### Build Options (Explicit)

- `vector/Dockerfile` does not compile from source; it unpacks the
  upstream `x86_64-unknown-linux-musl` release tarball.
- Packaged binary path is `${TARGET_PREFIX}/bin/vector`.
- Because the upstream release binary is reused, source/transform/sink
  availability is determined by that release.

### Runtime/Packaging Snapshot

- Version/platform metadata is captured in
  [Runtime Introspection Output](#runtime-introspection-output) with
  `vector --version`.
- Component inventory is captured with `vector list`.

## Allowed Target-Specific Variations

- This target intentionally repackages the upstream musl release tarball
  instead of compiling Vector from source inside this repository.
- The approved release artifact for this target is `bin/vector`.
- The available sources, transforms, sinks, and enrichment tables are
  intentionally defined by the upstream release selected in
  `metadata.json`.

## How to Verify

> [!NOTE]
> Outputs are under `.out/vector/`. Override with `BUILD_OUTPUT_DEST`.

```bash
./.out/vector/bin/vector list
```

## Runtime Introspection Output

### vector --version

```text
vector 0.53.0 (x86_64-unknown-linux-musl 2b51b40 2026-01-27 21:46:39.386326724)
```

### vector list

```text
Sources:
- amqp
- apache_metrics
- aws_ecs_metrics
- aws_kinesis_firehose
- aws_s3
- aws_sqs
- datadog_agent
- demo_logs
- dnstap
- docker_logs
- eventstoredb_metrics
- exec
- file
- file_descriptor
- fluent
- gcp_pubsub
- heroku_logs
- host_metrics
- http
- http_client
- http_server
- internal_logs
- internal_metrics
- journald
- kafka
- kubernetes_logs
- logstash
- mongodb_metrics
- mqtt
- nats
- nginx_metrics
- okta
- opentelemetry
- postgresql_metrics
- prometheus_pushgateway
- prometheus_remote_write
- prometheus_scrape
- pulsar
- redis
- socket
- splunk_hec
- static_metrics
- statsd
- stdin
- syslog
- unit_test
- unit_test_stream
- vector
- websocket

Transforms:
- aggregate
- aws_ec2_metadata
- dedupe
- exclusive_route
- filter
- incremental_to_absolute
- log_to_metric
- lua
- metric_to_log
- reduce
- remap
- route
- sample
- tag_cardinality_limit
- throttle
- trace_to_log
- window

Sinks:
- amqp
- appsignal
- aws_cloudwatch_logs
- aws_cloudwatch_metrics
- aws_kinesis_firehose
- aws_kinesis_streams
- aws_s3
- aws_sns
- aws_sqs
- axiom
- azure_blob
- azure_monitor_logs
- blackhole
- clickhouse
- console
- databend
- datadog_events
- datadog_logs
- datadog_metrics
- datadog_traces
- doris
- elasticsearch
- file
- gcp_chronicle_unstructured
- gcp_cloud_storage
- gcp_pubsub
- gcp_stackdriver_logs
- gcp_stackdriver_metrics
- greptimedb
- greptimedb_logs
- greptimedb_metrics
- honeycomb
- http
- humio_logs
- humio_metrics
- influxdb_logs
- influxdb_metrics
- kafka
- keep
- logdna
- loki
- mezmo
- mqtt
- nats
- new_relic
- opentelemetry
- papertrail
- postgres
- prometheus_exporter
- prometheus_remote_write
- pulsar
- redis
- sematext_logs
- sematext_metrics
- socket
- splunk_hec_logs
- splunk_hec_metrics
- statsd
- unit_test
- unit_test_stream
- vector
- webhdfs
- websocket
- websocket_server

Enrichment tables:
- file
- geoip
- memory
- mmdb
```
