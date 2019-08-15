package sns

import (
	"testing"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/beatlabs/patron/trace"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_NewPublisher(t *testing.T) {
	cfg, err := NewConfig(session.New())
	require.NoError(t, err)

	p := NewPublisher(*cfg)
	assert.IsType(t, &sns.SNS{}, p.sns)
	assert.Equal(t, p.component, trace.SNSPublisherComponent)
	assert.Equal(t, p.tag, ext.SpanKindProducer)
}

func Test_Publisher_publishOpName(t *testing.T) {
	component := "component"
	p := &TracedPublisher{
		component: component,
	}

	msg, err := NewMessageBuilder().Build()
	require.NoError(t, err)

	assert.Equal(t, "component publish:unknown", p.publishOpName(*msg))
}
