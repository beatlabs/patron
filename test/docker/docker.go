package docker

import (
	"errors"
	"time"

	"github.com/ory/dockertest"
)

// Runtime wraps dockertest functionality into a reusable component.
type Runtime struct {
	expiration time.Duration
	pool       *dockertest.Pool
	resources  []*dockertest.Resource
}

// NewRuntime constructor.
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

// Pool getter.
func (b *Runtime) Pool() *dockertest.Pool {
	return b.pool
}

// Expiration getter.
func (b *Runtime) Expiration() time.Duration {
	return b.expiration
}

// AppendResources to the internal resources list.
func (b *Runtime) AppendResources(r *dockertest.Resource) {
	b.resources = append(b.resources, r)
}

// Teardown all resources in the opposite order of their creation.
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
