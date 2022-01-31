# Observability

## Metrics and Tracing

Tracing and metrics are provided by Jaeger's implementation of the OpenTracing project and Prometheus.
Every component has been integrated with the above library and produces traces and metrics.
Metrics are can be scraped via the default HTTP component at the `/metrics` route for Prometheus.  
Traces will be sent to a Jaeger agent, which can be setup through environment variables mentioned in the config section.    
Sane defaults are applied for making the use easy.  
The `component` and `client` packages implement capturing and propagating of metrics and traces.

## Prometheus Exemplars

[OpenTracing](https://opentracing.io) compatible tracing systems such as [Grafana Tempo](https://grafana.com/oss/tempo/)
can work with [Prometheus Exemplars](https://grafana.com/docs/grafana/latest/basics/exemplars/).

Below are prerequisites for enabling exemplars:

- Use Prometheus Go client library version 1.4.0 or above.
- Use the new `ExemplarObserver` for `Histogram` or `ExemplarAdder` for `Counter` 
  because the original interfaces has not been changed for the backward compatibility.
- Use `ObserveWithExemplar` or `AddWithExemplar` methods noting the `TraceID` key — it is needed later to configure 
  Grafana, so that it knows which label to use to retrieve the `TraceID`

An example of enabling exemplars in an already instrumented Go application can be found [here](../trace/metric.go)
where exemplars are enabled for `Histogram` and `Counter` metrics.

The result of the above steps is attached trace IDs to metrics via exemplars.
When querying `/metrics` endpoint `curl -H "Accept: application/openmetrics-text"  <endpoint>:<port>/metrics`
exemplars will be present in metrics entry after `#` in [Open Metrics](https://github.com/OpenObservability/OpenMetrics/blob/main/specification/OpenMetrics.md#exemplars-1) format.