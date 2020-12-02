# gRPC

The gRPC component can be used to create a gRPC server. 
To enable observability, it injects unary and stream interceptors.

As the server implements the Patron `component` interface, it also handles graceful shutdown via the passed context.

Setting up a gRPC component is done via the Builder (which follows the builder pattern), and supports various configuration values in the form of the `grpc.ServerOption` struct during setup.

Check out the [examples/](/examples) folder for an hands-on tutorial on setting up a server and working with gRPC in Patron.