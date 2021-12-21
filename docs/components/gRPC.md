# gRPC

The gRPC component can be used to create a gRPC server. 
To enable observability, it injects unary and stream interceptors.

As the server implements the Patron `component` interface, it also handles graceful shutdown via the passed context.

Setting up a gRPC component is done via the Builder (which follows the builder pattern), and supports various configuration values in the form of the `grpc.ServerOption` struct during setup.

Check out the [examples/](/examples) folder for an hands-on tutorial on setting up a server and working with gRPC in Patron.

## Metrics

The following metrics are automatically provided when using `WithTrace()`:
* `component_grpc_handled_total`
* `component_grpc_handled_seconds`

Example of the associated labels: `grpc_code="OK"`, `grpc_method="CreateMyEvent"`, `grpc_service="myservice.Service"`, `grpc_type="unary"`.