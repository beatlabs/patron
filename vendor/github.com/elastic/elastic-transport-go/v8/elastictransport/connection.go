// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package elastictransport

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net/url"
	"slices"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

var (
	defaultResurrectTimeoutInitial      = 60 * time.Second
	defaultResurrectTimeoutFactorCutoff = 5

	// zero-allocation interface assertions
	_ ConnectionPool          = (*singleConnectionPool)(nil)
	_ ConnectionPool          = (*statusConnectionPool)(nil)
	_ UpdatableConnectionPool = (*statusConnectionPool)(nil)
	_ CloseableConnectionPool = (*statusConnectionPool)(nil)
	_ ConnectionPool          = (*synchronizedPool)(nil)
	_ CloseableConnectionPool = (*synchronizedPool)(nil)
	_ connectionable          = (*synchronizedPool)(nil)
	_ ConnectionPool          = (*synchronizedUpdatablePool)(nil)
	_ CloseableConnectionPool = (*synchronizedUpdatablePool)(nil)
	_ UpdatableConnectionPool = (*synchronizedUpdatablePool)(nil)
	_ Selector                = (*roundRobinSelector)(nil)
)

// Selector defines the interface for selecting connections from the pool.
type Selector interface {
	Select([]*Connection) (*Connection, error)
}

// ConnectionPool defines the interface for the connection pool.
type ConnectionPool interface {
	Next() (*Connection, error)  // Next returns the next available connection.
	OnSuccess(*Connection) error // OnSuccess reports that the connection was successful.
	OnFailure(*Connection) error // OnFailure reports that the connection failed.
	URLs() []*url.URL            // URLs returns the list of URLs of available connections.
}

type UpdatableConnectionPool interface {
	Update([]*Connection) error // Update injects newly found nodes in the cluster.
}

// ConcurrentSafeConnectionPool marks a custom pool as safe for concurrent use
// by the transport. Pools returned by ConnectionPoolFunc that implement this
// interface are not wrapped by synchronizedPool.
type ConcurrentSafeConnectionPool interface {
	ConnectionPool
	ConcurrentSafe()
}

// CloseableConnectionPool defines the interface for the connection pool that can be closed.
type CloseableConnectionPool interface {
	Close(context.Context) error
}

// synchronizedPool wraps a ConnectionPool with a mutex to serialize all method
// calls. Used to wrap user-provided pools from ConnectionPoolFunc that may not
// be safe for concurrent use.
//
// Use newSynchronizedPool to construct; it returns a synchronizedUpdatablePool
// when the inner pool implements UpdatableConnectionPool.
type synchronizedPool struct {
	mu   sync.Mutex
	pool ConnectionPool
}

// synchronizedUpdatablePool extends synchronizedPool for inner pools that
// support in-place updates. Discovery will prefer Update() over full pool
// replacement when this interface is present.
type synchronizedUpdatablePool struct {
	synchronizedPool
}

// newSynchronizedPool wraps pool in a synchronized adapter by default.
// ConcurrentSafeConnectionPool implementations are returned as-is.
// If pool implements UpdatableConnectionPool the returned value will too, so
// discovery can update the pool in place rather than replacing it.
func newSynchronizedPool(pool ConnectionPool) ConnectionPool {
	if _, ok := pool.(ConcurrentSafeConnectionPool); ok {
		return pool
	}

	if _, ok := pool.(UpdatableConnectionPool); ok {
		return &synchronizedUpdatablePool{synchronizedPool{pool: pool}}
	}
	return &synchronizedPool{pool: pool}
}

func (sp *synchronizedPool) Next() (*Connection, error) {
	sp.mu.Lock()
	defer sp.mu.Unlock()
	return sp.pool.Next()
}

func (sp *synchronizedPool) OnSuccess(c *Connection) error {
	sp.mu.Lock()
	defer sp.mu.Unlock()
	return sp.pool.OnSuccess(c)
}

func (sp *synchronizedPool) OnFailure(c *Connection) error {
	sp.mu.Lock()
	defer sp.mu.Unlock()
	return sp.pool.OnFailure(c)
}

func (sp *synchronizedPool) URLs() []*url.URL {
	sp.mu.Lock()
	defer sp.mu.Unlock()
	return sp.pool.URLs()
}

func (sp *synchronizedPool) Close(ctx context.Context) error {
	sp.mu.Lock()
	cp, ok := sp.pool.(CloseableConnectionPool)
	sp.mu.Unlock()
	if !ok {
		return nil
	}
	// Avoid holding the wrapper lock during close; in-flight callbacks also take
	// this lock and may need to complete before the inner pool can fully close.
	return cp.Close(ctx)
}

func (sp *synchronizedPool) connections() []*Connection {
	sp.mu.Lock()
	defer sp.mu.Unlock()
	if cp, ok := sp.pool.(connectionable); ok {
		return cp.connections()
	}
	return nil
}

func (sp *synchronizedUpdatablePool) Update(conns []*Connection) error {
	sp.mu.Lock()
	defer sp.mu.Unlock()
	up, ok := sp.pool.(UpdatableConnectionPool)
	if !ok {
		return fmt.Errorf("inner pool %T does not implement UpdatableConnectionPool", sp.pool)
	}
	return up.Update(conns)
}

// Connection represents a connection to a node.
type Connection struct {
	sync.Mutex

	URL       *url.URL
	IsDead    bool
	DeadSince time.Time
	Failures  int

	ID         string
	Name       string
	Roles      []string
	Attributes map[string]interface{}
}

func (c *Connection) Cmp(connection *Connection) bool {
	if c.URL.Hostname() == connection.URL.Hostname() {
		if c.URL.Port() == connection.URL.Port() {
			return c.URL.Path == connection.URL.Path
		}
	}
	return false
}

type singleConnectionPool struct {
	connection *Connection

	metrics *metrics
}

type statusConnectionPool struct {
	sync.Mutex

	live     []*Connection // List of live connections
	dead     []*Connection // List of dead connections
	selector Selector

	metrics *metrics

	resurrectWaitGroup sync.WaitGroup
	closeC             chan struct{}
	closeDone          uint32
}

type roundRobinSelector struct {
	curr uint64 // Atomic counter for next connection index
}

func newRoundRobinSelector() *roundRobinSelector {
	return &roundRobinSelector{curr: math.MaxUint64}
}

// NewConnectionPool creates and returns a default connection pool.
func NewConnectionPool(conns []*Connection, selector Selector) (ConnectionPool, error) {
	if len(conns) == 1 {
		return &singleConnectionPool{connection: conns[0]}, nil
	}
	if selector == nil {
		selector = newRoundRobinSelector()
	}
	return &statusConnectionPool{live: slices.Clone(conns), selector: selector, closeC: make(chan struct{})}, nil
}

// Next returns the connection from pool.
func (cp *singleConnectionPool) Next() (*Connection, error) {
	return cp.connection, nil
}

// OnSuccess is a no-op for single connection pool.
func (cp *singleConnectionPool) OnSuccess(c *Connection) error { return nil }

// OnFailure is a no-op for single connection pool.
func (cp *singleConnectionPool) OnFailure(c *Connection) error { return nil }

// URLs returns the list of URLs of available connections.
func (cp *singleConnectionPool) URLs() []*url.URL { return []*url.URL{cp.connection.URL} }

func (cp *singleConnectionPool) connections() []*Connection { return []*Connection{cp.connection} }

// Next returns a connection from pool, or an error.
func (cp *statusConnectionPool) Next() (*Connection, error) {
	if cp.isClosed() {
		return nil, errors.New("connection pool is closed")
	}
	cp.Lock()
	defer cp.Unlock()

	// Return next live connection
	if len(cp.live) > 0 {
		return cp.selector.Select(cp.live)
	} else if len(cp.dead) > 0 {
		// No live connection is available, resurrect one of the dead ones.
		c := cp.dead[len(cp.dead)-1]
		cp.dead = slices.Delete(cp.dead, len(cp.dead)-1, len(cp.dead))
		c.Lock()
		defer c.Unlock()
		err := cp.resurrect(c, false)
		if err != nil {
			return nil, err
		}
		return c, nil
	}
	return nil, errors.New("no connection available")
}

// OnSuccess marks the connection as successful.
func (cp *statusConnectionPool) OnSuccess(c *Connection) error {
	// Short-circuit for live connection
	c.Lock()
	if !c.IsDead {
		c.Unlock()
		return nil
	}
	c.Unlock()

	cp.Lock()
	defer cp.Unlock()

	c.Lock()
	defer c.Unlock()

	if !c.IsDead {
		return nil
	}

	c.markAsHealthy()
	return cp.resurrect(c, true)
}

// OnFailure marks the connection as failed.
func (cp *statusConnectionPool) OnFailure(c *Connection) error {
	cp.Lock()
	defer cp.Unlock()

	c.Lock()

	if c.IsDead {
		if debugLogger != nil {
			_ = debugLogger.Logf("Already removed %s\n", c.URL)
		}
		c.Unlock()
		return nil
	}

	if debugLogger != nil {
		_ = debugLogger.Logf("Removing %s...\n", c.URL)
	}
	c.markAsDead()
	cp.scheduleResurrect(c)
	c.Unlock()

	// Check if connection exists in the list, return error if not.
	index := -1
	for i, conn := range cp.live {
		if conn == c {
			index = i
		}
	}
	if index < 0 {
		return errors.New("connection not in live list")
	}

	// Push item to dead list and sort slice by number of failures
	cp.dead = append(cp.dead, c)
	sort.Slice(cp.dead, func(i, j int) bool {
		c1 := cp.dead[i]
		c2 := cp.dead[j]

		res := c1.Failures > c2.Failures
		return res
	})

	cp.live = slices.Delete(cp.live, index, index+1)

	return nil
}

// Update merges the existing live and dead connections with the latest nodes discovered from the cluster.
func (cp *statusConnectionPool) Update(connections []*Connection) error {
	cp.Lock()
	defer cp.Unlock()

	if len(connections) == 0 {
		return errors.New("no connections provided, connection pool left untouched")
	}

	// Remove hosts that are no longer in the new list of connections
	for i := 0; i < len(cp.live); i++ {
		found := false
		for _, c := range connections {
			if cp.live[i].Cmp(c) {
				found = true
				break
			}
		}

		if !found {
			cp.live = slices.Delete(cp.live, i, i+1)
			i--
		}
	}

	// Remove hosts that are no longer in the dead list of connections
	for i := 0; i < len(cp.dead); i++ {
		found := false
		for _, c := range connections {
			if cp.dead[i].Cmp(c) {
				found = true
				break
			}
		}

		if !found {
			cp.dead = slices.Delete(cp.dead, i, i+1)
			i--
		}
	}

	// Add new connections that are not already in the live or dead list
	for _, c := range connections {
		found := false
		for _, conn := range cp.live {
			if conn.Cmp(c) {
				found = true
				break
			}
		}
		for _, conn := range cp.dead {
			if conn.Cmp(c) {
				found = true
				break
			}
		}
		if !found {
			cp.live = append(cp.live, c)
		}
	}

	return nil
}

// URLs returns the list of URLs of available connections.
func (cp *statusConnectionPool) URLs() []*url.URL {
	var urls []*url.URL

	cp.Lock()
	defer cp.Unlock()

	for _, c := range cp.live {
		urls = append(urls, c.URL)
	}

	return urls
}

// Close closes the connection pool. After which it is no longer possible to use the pool.
// It will wait until all pending resurrection routines have successfully been cancelled.
// Close should only be called once, otherwise it will return an error.
func (cp *statusConnectionPool) Close(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}

	if atomic.CompareAndSwapUint32(&cp.closeDone, 0, 1) {
		close(cp.closeC)

		done := make(chan struct{})
		go func() {
			defer close(done)
			cp.resurrectWaitGroup.Wait()
		}()

		select {
		case <-done:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	} else {
		return errors.New("connection pool is already closed")
	}

}

func (cp *statusConnectionPool) isClosed() bool {
	select {
	case <-cp.closeC:
		return true
	default:
		return false
	}
}

func (cp *statusConnectionPool) connections() []*Connection {
	cp.Lock()
	defer cp.Unlock()
	conns := make([]*Connection, 0, len(cp.live)+len(cp.dead))
	conns = append(conns, cp.live...)
	conns = append(conns, cp.dead...)
	return conns
}

// resurrect adds the connection to the list of available connections.
// When removeDead is true, it also removes it from the dead list.
// The calling code is responsible for locking.
func (cp *statusConnectionPool) resurrect(c *Connection, removeDead bool) error {
	if debugLogger != nil {
		_ = debugLogger.Logf("Resurrecting %s\n", c.URL)
	}

	c.markAsLive()
	cp.live = append(cp.live, c)

	if removeDead {
		index := -1
		for i, conn := range cp.dead {
			if conn == c {
				index = i
			}
		}
		if index >= 0 {
			cp.dead = slices.Delete(cp.dead, index, index+1)
		}
	}

	return nil
}

// scheduleResurrect schedules the connection to be resurrected.
func (cp *statusConnectionPool) scheduleResurrect(c *Connection) {
	factor := math.Min(float64(c.Failures-1), float64(defaultResurrectTimeoutFactorCutoff))
	timeout := time.Duration(defaultResurrectTimeoutInitial.Seconds() * math.Exp2(factor) * float64(time.Second))
	if debugLogger != nil {
		_ = debugLogger.Logf("Resurrect %s (failures=%d, factor=%1.1f, timeout=%s) in %s\n", c.URL, c.Failures, factor, timeout, c.DeadSince.Add(timeout).Sub(time.Now().UTC()).Truncate(time.Second))
	}

	cp.resurrectWaitGroup.Add(1)
	go func() {
		defer cp.resurrectWaitGroup.Done()
		timer := time.NewTimer(timeout)
		defer timer.Stop()

		select {
		case <-timer.C:
			cp.Lock()
			defer cp.Unlock()

			c.Lock()
			defer c.Unlock()

			if !c.IsDead {
				if debugLogger != nil {
					_ = debugLogger.Logf("Already resurrected %s\n", c.URL)
				}
				return
			}

			err := cp.resurrect(c, true)
			if err != nil {
				return
			}
		case <-cp.closeC:
			return
		}
	}()
}

// Select returns the connection in a round-robin fashion.
func (s *roundRobinSelector) Select(conns []*Connection) (*Connection, error) {
	index := atomic.AddUint64(&s.curr, 1) % uint64(len(conns))
	return conns[int(index)], nil
}

// markAsDead marks the connection as dead.
func (c *Connection) markAsDead() {
	c.IsDead = true
	if c.DeadSince.IsZero() {
		c.DeadSince = time.Now().UTC()
	}
	c.Failures++
}

// markAsLive marks the connection as alive.
func (c *Connection) markAsLive() {
	c.IsDead = false
}

// markAsHealthy marks the connection as healthy.
func (c *Connection) markAsHealthy() {
	c.IsDead = false
	c.DeadSince = time.Time{}
	c.Failures = 0
}

// String returns a readable connection representation.
func (c *Connection) String() string {
	c.Lock()
	defer c.Unlock()
	return fmt.Sprintf("<%s> dead=%v failures=%d", c.URL, c.IsDead, c.Failures)
}
