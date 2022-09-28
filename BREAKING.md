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
