# Breaking Changes Migration Guide

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

## v1.0.0

### Instantiation of patron service

The instantiation and initialisation of the patron main service has been moved from `builder pattern` to `functional options pattern`.
The optional configuration parameters used for the builder in previous versions can now be passed as Options to the service constructor.

#### Creating a patron instance without additional options

```go
package main

import (
	"context"
	"github.com/beatlabs/patron"
	"log"
)

func main() {
	name := "service-name"
	version := "version"

	svc, err := patron.New(name, version)
	if err != nil {
		log.Fatalf("failed to create patron service due to : %s", err)
	}

	ctx := context.Background()
	err = svc.Run(ctx)
	if err != nil {
		log.Fatalf("failed to run service %s", err)
	}
}
```

#### Creating a patron instance with components

```go
amqp, err := patronamqp.New(url, queue, amqpCmp.Process, patronamqp.Retry(10, 1*time.Second))
if err != nil {
    log.Fataln("failed to create amqp component due: %s",err)
}

grpc,err := patrongrpc.New(1234)
if err != nil {
    log.Fatalf("failed to create gRPC component: %v", err)
}

svc, err := patron.New(name, version, patron.Components(amqp,grpc))
    if err != nil {
    log.Fatalf("failed to create patron service due to : %s", err)
}

ctx := context.Background()
err = svc.Run(ctx)
if err != nil {
    log.Fatalf("failed to run service %s", err)
}

```
Check the [examples directory](./examples) for complete examples of detailed usage.

#### Creating a patron instance with a custom logger 

```go
package main

import (
	"context"
	"github.com/beatlabs/patron"
	"github.com/beatlabs/patron/log/std"
	"log"
)

func main() {
	name := "service-name"
	version := "version"

	logger := std.New(os.Stderr, log.DebugLevel, map[string]interface{}{"env": "staging"})
	
	svc, err := patron.New(name, version, patron.Logger(logger))
	if err != nil {
		log.Fatalf("failed to create patron service due to : %s", err)
	}

	ctx := context.Background()
	err = svc.Run(ctx)
	if err != nil {
		log.Fatalf("failed to run service %s", err)
	}
}
```

#### Creating a patron instance with default text logger option

```go
package main

import (
	"context"
	"github.com/beatlabs/patron"
	"log"
)

func main() {
	name := "service-name"
	version := "version"
	
	svc, err := patron.New(name, version, patron.TextLogger())
	if err != nil {
		log.Fatalf("failed to create patron service due to : %s", err)
	}

	ctx := context.Background()
	err = svc.Run(ctx)
	if err != nil {
		log.Fatalf("failed to run service %s", err)
	}
}
```

#### Creating a patron instance with custom log fields option

```go
package main

import (
	"context"
	"github.com/beatlabs/patron"
	"log"
)

func main() {
	name := "service-name"
	version := "version"
	
	svc, err := patron.New(name, version, patron.LogFields(map[string]interface{}{"key": "value"}))
	if err != nil {
		log.Fatalf("failed to create patron service due to : %s", err)
	}

	ctx := context.Background()
	err = svc.Run(ctx)
	if err != nil {
		log.Fatalf("failed to run service %s", err)
	}
}
```

#### Creating a patron instance with SIGHUP handler option

```go
package main

import (
	"context"
	"github.com/beatlabs/patron"
	"log"
	"os"
)

func main() {
	name := "service-name"
	version := "version"

	sig := func() {
		log.Println("exit gracefully...")
		os.Exit(0)
	}
	
	svc, err := patron.New(name, version, patron.SIGHUP(sig))
	if err != nil {
		log.Fatalf("failed to create patron service due to : %s", err)
	}

	ctx := context.Background()
	err = svc.Run(ctx)
	if err != nil {
		log.Fatalf("failed to run service %s", err)
	}
}
```

#### Creating a patron instance with a custom HTTP Router

```go
package main

import (
	"context"
	"github.com/beatlabs/patron"
	"log"
	"net/http"
	v2 "github.com/beatlabs/patron/component/http/v2"
	"github.com/beatlabs/patron/component/http/v2/router/httprouter"
)

func main() {
	name := "service-name"
	version := "version"
	
	var routes v2.Routes
	routes.Append(v2.NewGetRoute("/hello", hello))

	rr, err := routes.Result()
	if err != nil {
		log.Fatalf("failed to create routes: %v", err)
	}

	router, err := httprouter.New(httprouter.Routes(rr...))
	if err != nil {
		log.Fatalf("failed to create http router: %v", err)
	}

	svc, err := patron.New(name, version, patron.Router(router))
	if err != nil {
		log.Fatalf("failed to create patron service due to : %s", err)
	}

	ctx := context.Background()
	err = svc.Run(ctx)
	if err != nil {
		log.Fatalf("failed to run service %s", err)
	}
}

func hello(rw http.ResponseWriter, _ *http.Request) {
	rw.WriteHeader(http.StatusCreated)
	_, _ = rw.Write([]byte("hello!"))
	return
}
```

### Instantiation of v2 GRPC Component 

The instantiation and initialisation of the GRPC component has been moved from `builder pattern` to `functional options pattern`.
The configuration parameters used for the builder in previous versions can now be passed as Options to the component constructor.
An example of configuring a client for a service (e.g. `SQS`) and plugging it on a patron component is demonstrated below:

#### Creating a new grpc component without additional options
```go
package main

import (
	"github.com/beatlabs/patron/component/grpc"
	"log"
)

func main(){
	port := 5000
	comp,err := grpc.New(port)
	if err != nil{
		log.Fatalf("failed to create new grpc component due: %s",err)
    }
}

```

#### Creating a new grpc component with grpc server options

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
	comp,err := patrongrpc.New(port, patrongrpc.ServerOptions(grpc.ConnectionTimeout(1*time.Second)))
	if err != nil{
		log.Fatalf("failed to create new grpc component due: %s",err)
	}
}
```

#### Creating a new grpc component with reflection

```go
package main

import (
	patrongrpc "github.com/beatlabs/patron/component/grpc"
	"log"
)

func main(){
	port := 5000
	comp,err := patrongrpc.New(port, patrongrpc.Reflection())
	if err != nil{
		log.Fatalf("failed to create new grpc component due: %s",err)
	}
}
```