package sns

import (
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/stretchr/testify/assert"
)

func Test_NewConfig(t *testing.T) {
	type args struct {
		sess *session.Session
		cfgs []*aws.Config
	}
	testCases := []struct {
		desc        string
		args        args
		expectedErr error
	}{
		{
			desc:        "Missing session",
			args:        args{},
			expectedErr: errors.New("please provide an AWS session"),
		},
		{
			desc: "Session only",
			args: args{
				sess: session.Must(session.NewSession()),
			},
		},
		{
			desc: "Session and cfgs",
			args: args{
				sess: session.Must(session.NewSession()),
				cfgs: []*aws.Config{&aws.Config{}},
			},
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			cfg, err := NewConfig(tC.args.sess, tC.args.cfgs...)
			if tC.expectedErr != nil {
				assert.EqualError(t, err, tC.expectedErr.Error())
				assert.Nil(t, cfg)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tC.args.sess, cfg.sess)
				assert.Equal(t, tC.args.cfgs, cfg.cfgs)
			}
		})
	}
}
