# Observability

## Metrics and Tracing

Tracing and metrics are provided by Jaeger's implementation of the OpenTracing project and Prometheus.
Every component has been integrated with the above library and produces traces and metrics.
Metrics are can be scraped via the default HTTP component at the `/metrics` route for Prometheus.  
Traces will be sent to a Jaeger agent, which can be setup through environment variables mentioned in the config section.    
Sane defaults are applied for making the use easy.  
The `component` and `client` packages implement capturing and propagating of metrics and traces.