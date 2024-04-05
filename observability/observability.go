// package observability is based on OpenTelemetry.
package observability

// Setup initializes OpenTelemetry's traces and metrics.
func Setup(name string) error {
	err := setupMeter(name)
	if err != nil {
		return err
	}
	return setupTracing(name)
}
