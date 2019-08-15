package sns

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/beatlabs/patron/errors"
)

// Config struct for the SNS publisher.
type Config struct {
	sess *session.Session
	cfgs []*aws.Config
}

// NewConfig tries to create a new config for the SNS publisher and returns an error if some data is invalid.
func NewConfig(sess *session.Session, cfgs ...*aws.Config) (*Config, error) {
	if sess == nil {
		return nil, errors.New("please provide an AWS session")
	}

	return &Config{
		sess: sess,
		cfgs: cfgs,
	}, nil
}
