# Breaking Changes Migration Guide

## v0.75.0

The `cache` package has introduced the `context.Context` as the first argument in each method and removed it from the constructors.

## v0.74.0

### Instantiation of patron service

The instantiation and initialisation of the patron main service has been moved from `builder pattern` to `functional options pattern`.
The optional configuration parameters used for the builder in previous versions can now be passed as Options to the service constructor.
Check the [examples directory](./examples) for complete examples of detailed usage.

Types, github.com/beatlabs/patron.Builder and  github.com/beatlabs/patron.Option discontinued.

#### Creating a patron instance with components

##### v0.73.0 and before

```go
svc, err := patron.New(name, version)
if err != nil {
    log.Fatalf("failed to create patron service due to : %s", err)
}

ctx := context.Background()
err = svc.WithComponents(ampq,grpc).Run(ctx)
if err != nil {
    log.Fatalf("failed to run service %s", err)
}
```

##### v0.74.0

```go
svc, err := patron.New(name, version, patron.WithComponents(amqp,grpc))
if err != nil {
    log.Fatalf("failed to create patron service due to : %s", err)
}

ctx := context.Background()
err = svc.Run(ctx)
if err != nil {
    log.Fatalf("failed to run service %s", err)
}

```

#### Creating a patron instance with SIGHUP handler option

##### v0.73.0 and before

```go
svc, err := patron.New(name, version)
if err != nil {
    log.Fatalf("failed to create patron service due to : %s", err)
}

ctx := context.Background()
err = svc.WithSIGHUP(sighup).Run(ctx)
if err != nil {
    log.Fatalf("failed to run service %s", err)
}
```

##### v0.74.0

```go
svc, err := patron.New(name, version, patron.WithSIGHUP(sighup))
if err != nil {
    log.Fatalf("failed to create patron service due to : %s", err)
}

ctx := context.Background()
err = svc.Run(ctx)
if err != nil {
    log.Fatalf("failed to run service %s", err)
}

```

#### Creating a patron instance with a custom HTTP Router

##### v0.73.0 and before

```go
svc, err := patron.New(name, version)
if err != nil {
    log.Fatalf("failed to create patron service due to : %s", err)
}

ctx := context.Background()
err = svc.WithRouter(router).Run(ctx)
if err != nil {
    log.Fatalf("failed to run service %s", err)
}
```

##### v0.74.0

```go
svc, err := patron.New(name, version, patron.WithRouter(router))
if err != nil {
    log.Fatalf("failed to create patron service due to : %s", err)
}

ctx := context.Background()
err = svc.Run(ctx)
if err != nil {
    log.Fatalf("failed to run service %s", err)
}

```

### Instantiation of v2 GRPC Component

The instantiation and initialisation of the GRPC component has been moved from `builder pattern` to `functional options pattern`.
The configuration parameters used for the builder in previous versions can now be passed as Options to the component constructor.

##### v0.73.0 and before

```go
package main

import (
 patrongrpc "github.com/beatlabs/patron/component/grpc"
 "google.golang.org/grpc"
 "log"
 "time"
)

func main(){
 port := 5000
 builder,err := grpc.WithOptions(grpc.ConnectionTimeout(1*time.Second)).WithReflection().New(port)
 if err != nil{
  log.Fatalf("failed to create new grpc builder due: %s",err)
 }

 comp,err := builder.Create()
 if err != nil{
  log.Fatalf("failed to create grpc component due: %s",err)
 }
}
```

##### v0.74.0

```go
package main

import (
 patrongrpc "github.com/beatlabs/patron/component/grpc"
 "log"
 "google.golang.org/grpc"
 "time"
)

func main(){
 port := 5000
 comp,err := patrongrpc.New(port, patrongrpc.WithServerOptions(grpc.ConnectionTimeout(1*time.Second)),patrongrpc.WithReflection())
 if err != nil{
  log.Fatalf("failed to create new grpc component due: %s",err)
 }
}
```

#### Changes to method Signatures

In package `github.com/beatlabs/patron/component/http/v2/router/httprouter`,

- `func EnableAppNameHeaders(name, version string) OptionFunc` renamed to `func WithAppNameHeaders(name, version string) OptionFunc`

- `func EnableExpVarProfiling() OptionFunc` renamed to `func WithExpVarProfiling() OptionFunc`

All names of other option functions for `components/clients` prefixed by `With`

i.e.

considering package `github.com/beatlabs/patron/client/ampq/v2`

option function `func Config(cfg amqp.Config) OptionFunc` is renamed to `func Config(cfg amqp.Config) OptionFunc`

## v0.73.0

### Migrating from `aws-sdk-go` v1 to v2

For leveraging the AWS patron components updated to `aws-sdk-go` v2, the client initialization should be modified. In v2 the [session](https://docs.aws.amazon.com/sdk-for-go/api/aws/session) package was replaced with a simple configuration system provided by the [config](https://pkg.go.dev/github.com/aws/aws-sdk-go-v2/config) package.

Options such as the AWS region and endpoint to be used can be mentioned during configuration loading.

Endpoint resolvers are used to specify custom endpoints for a particular service and region.

The AWS client configured can be plugged in to the respective patron component on initialization, in the same way its predecessor did in earlier patron versions.

An example of configuring a client for a service (e.g. `SQS`) and plugging it on a patron component is demonstrated below:

```go
import (
 "context"

 "github.com/aws/aws-sdk-go-v2/aws"
 "github.com/aws/aws-sdk-go-v2/config"
 "github.com/aws/aws-sdk-go-v2/credentials"
 "github.com/aws/aws-sdk-go-v2/service/sqs"
)

const (
 awsRegion      = "eu-west-1"
 awsID          = "test"
 awsSecret      = "test"
 awsToken       = "token"
 awsSQSEndpoint = "http://localhost:4566"
)

func main() {
 ctx := context.Background()

 sqsAPI, err := createSQSAPI(awsSQSEndpoint)
 if err != nil {// handle error}

 sqsCmp, err := createSQSComponent(sqsAPI) // implementation ommitted
 if err != nil {// handle error}
 
 err = service.WithComponents(sqsCmp.cmp).Run(ctx)
 if err != nil {// handle error}
}

func createSQSAPI(endpoint string) (*sqs.Client, error) {
 customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
  if service == sqs.ServiceID && region == awsRegion {
   return aws.Endpoint{
    URL:           endpoint,
    SigningRegion: awsRegion,
   }, nil
  }
  // returning EndpointNotFoundError will allow the service to fallback to it's default resolution
  return aws.Endpoint{}, &aws.EndpointNotFoundError{}
 })

 cfg, err := config.LoadDefaultConfig(context.TODO(),
  config.WithRegion(awsRegion),
  config.WithEndpointResolverWithOptions(customResolver),
  config.WithCredentialsProvider(aws.NewCredentialsCache(credentials.NewStaticCredentialsProvider(awsID, awsSecret, awsToken))),
 )
 if err != nil {
  return nil, err
 }

 api := sqs.NewFromConfig(cfg)

 return api, nil
}
```

> A more detailed documentation on migrating can be found [here](https://aws.github.io/aws-sdk-go-v2/docs/migrating).
