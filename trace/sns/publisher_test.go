package sns

import (
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/stretchr/testify/require"
)

func Test_Publisher(t *testing.T) {
	var sess = session.Must(session.NewSession())
	cfg, err := NewConfig(
		sess,
		&aws.Config{
			Region:      aws.String("region"),
			Credentials: credentials.NewStaticCredentials("access-key-id", "secret", "token"),
		},
	)
	require.NoError(t, err)

	p, err := NewPublisher(*cfg)
	require.NoError(t, err)
	require.NotNil(t, p)
}
