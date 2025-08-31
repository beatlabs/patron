# AWS SNS client

Helper for constructing an AWS SDK v2 SNS client pre-wired with OpenTelemetry.

- Package: `github.com/beatlabs/patron/client/sns`
- Upstream: `github.com/aws/aws-sdk-go-v2/service/sns` and `go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-sdk-go-v2/otelaws`

## Usage

```go
import (
	"github.com/aws/aws-sdk-go-v2/aws"
	awscfg "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	patronsns "github.com/beatlabs/patron/client/sns"
)

cfg, err := awscfg.LoadDefaultConfig(ctx) // or custom
c := patronsns.NewFromConfig(cfg)
_, err = c.Publish(ctx, &sns.PublishInput{ Message: aws.String("hi"), TopicArn: aws.String(topicARN) })
```

- OTEL middleware is registered via `otelaws.AppendMiddlewares`.
