package docker

import (
	"errors"
	"time"

	"github.com/ory/dockertest"
)

type Runtime struct {
	expiration time.Duration
	pool       *dockertest.Pool
	resources  []*dockertest.Resource
}

func NewRuntime(expiration time.Duration) (*Runtime, error) {
	if expiration < 0 {
		return nil, errors.New("expiration value is negative")
	}

	pool, err := dockertest.NewPool("")
	if err != nil {
		return nil, err
	}
	pool.MaxWait = expiration

	return &Runtime{expiration: expiration, pool: pool}, nil
}

func (b *Runtime) Pool() *dockertest.Pool {
	return b.pool
}

func (b *Runtime) Expiration() time.Duration {
	return b.expiration
}

func (b *Runtime) AppendResources(r *dockertest.Resource) {
	b.resources = append(b.resources, r)
}

func (b *Runtime) Teardown() []error {
	ee := make([]error, 0)

	for i := len(b.resources) - 1; i >= 0; i-- {
		err := b.pool.Purge(b.resources[i])
		if err != nil {
			ee = append(ee, err)
		}
	}

	return ee
}
