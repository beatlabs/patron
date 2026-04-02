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
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

// Discoverable defines the interface for transports supporting node discovery.
type Discoverable interface {
	DiscoverNodes() error
}

// nodeInfo represents the information about node in a cluster.
//
// See: https://www.elastic.co/guide/en/elasticsearch/reference/current/cluster-nodes-info.html
type nodeInfo struct {
	ID         string
	Name       string
	URL        *url.URL
	Roles      []string `json:"roles"`
	Attributes map[string]interface{}
	HTTP       struct {
		PublishAddress string `json:"publish_address"`
	}
}

// DiscoverNodes reloads the client connections by fetching information from the cluster.
//
// Deprecated: Use DiscoverNodesContext instead
func (c *Client) DiscoverNodes() error {
	return c.DiscoverNodesContext(context.TODO())
}

// DiscoverNodesContext reloads the client connections by fetching information from the cluster.
func (c *Client) DiscoverNodesContext(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if c.isClosed() {
		return ErrClosed
	}

	var conns []*Connection

	nodes, err := c.getNodesInfo(ctx)
	if err != nil {
		if debugLogger != nil {
			_ = debugLogger.Logf("Error getting nodes info: %s\n", err)
		}
		return fmt.Errorf("discovery: get nodes: %w", err)
	}

	for _, node := range nodes {
		var (
			isMasterOnlyNode bool
		)

		roles := append(node.Roles[:0:0], node.Roles...)
		sort.Strings(roles)

		if len(roles) == 1 && roles[0] == "master" {
			isMasterOnlyNode = true
		}

		if debugLogger != nil {
			var skip string
			if isMasterOnlyNode {
				skip = "; [SKIP]"
			}
			_ = debugLogger.Logf("Discovered node [%s]; %s; roles=%s%s\n", node.Name, node.URL, node.Roles, skip)
		}

		// Skip master only nodes
		// TODO(karmi): Move logic to Selector?
		if isMasterOnlyNode {
			continue
		}

		conns = append(conns, &Connection{
			URL:        node.URL,
			ID:         node.ID,
			Name:       node.Name,
			Roles:      node.Roles,
			Attributes: node.Attributes,
		})
	}

	c.poolMu.Lock()
	defer c.poolMu.Unlock()

	if c.isClosed() {
		return ErrClosed
	}

	if len(conns) == 0 {
		if debugLogger != nil {
			_ = debugLogger.Logf("No eligible nodes discovered; pool left untouched\n")
		}
		return nil
	}

	if p, ok := c.pool.(UpdatableConnectionPool); ok {
		return p.Update(conns)
	}

	if c.poolFunc != nil {
		c.pool = newSynchronizedPool(c.poolFunc(conns, c.selector))
	} else {
		c.pool, err = NewConnectionPool(conns, c.selector)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *Client) getNodesInfo(ctx context.Context) ([]nodeInfo, error) {
	if len(c.urls) == 0 {
		return nil, errors.New("no URLs configured")
	}

	var (
		out    []nodeInfo
		scheme = c.urls[0].Scheme
	)

	var cancel context.CancelFunc
	if c.discoverNodeTimeout != nil {
		ctx, cancel = context.WithTimeout(ctx, *c.discoverNodeTimeout)
		defer cancel()
	}

	req, err := http.NewRequestWithContext(ctx, "GET", "/_nodes/http", nil)
	if err != nil {
		return out, err
	}

	pool := c.snapshotPool()
	conn, err := pool.Next()
	// TODO(karmi): If no connection is returned, fallback to original URLs
	if err != nil {
		return out, err
	}

	c.setReqURL(conn.URL, req)
	c.setReqAuth(conn.URL, req)
	c.setReqUserAgent(req)
	c.setReqGlobalHeader(req)

	res, err := c.roundTrip(req)
	if err != nil {
		return out, err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			if debugLogger != nil {
				_ = debugLogger.Logf("Error closing response body: %s\n", err)
			}
		}
	}(res.Body)

	if res.StatusCode > 200 {
		body, _ := io.ReadAll(res.Body)
		return out, fmt.Errorf("server error: %s: %s", res.Status, body)
	}

	var env map[string]json.RawMessage
	if err := json.NewDecoder(res.Body).Decode(&env); err != nil {
		return out, err
	}

	var nodes map[string]nodeInfo
	if err := json.Unmarshal(env["nodes"], &nodes); err != nil {
		return out, err
	}

	for id, node := range nodes {
		node.ID = id
		node.URL = c.getNodeURL(node, scheme)
		out = append(out, node)
	}

	return out, nil
}

func (c *Client) getNodeURL(node nodeInfo, scheme string) *url.URL {
	addr := node.HTTP.PublishAddress
	var host string

	// Elasticsearch publish_address may use "hostname/address:port" format.
	if idx := strings.IndexByte(addr, '/'); idx >= 0 {
		host = addr[:idx]
		addr = addr[idx+1:]
	}

	addrHost, port, err := net.SplitHostPort(addr)
	if err != nil {
		return &url.URL{Scheme: scheme, Host: addr}
	}

	if host == "" {
		host = addrHost
	}

	return &url.URL{
		Scheme: scheme,
		Host:   net.JoinHostPort(host, port),
	}
}

func (c *Client) scheduleDiscoverNodes(d time.Duration) {
	c.discoverWaitGroup.Add(1)
	go func() {
		defer c.discoverWaitGroup.Done()
		ticker := time.NewTicker(d)
		defer ticker.Stop()
		ctx, cancel := context.WithCancel(context.Background())
		wg := sync.WaitGroup{}

		for {
			select {
			case <-c.closeC:
				cancel()
				wg.Wait()
				return
			case <-ticker.C:
				wg.Add(1)
				go func() {
					defer wg.Done()
					ctx, cancel := context.WithTimeout(ctx, d)
					defer cancel()
					if err := c.DiscoverNodesContext(ctx); err != nil {
						if debugLogger != nil {
							_ = debugLogger.Logf("Error discovering nodes: %s\n", err)
						}
					}
				}()
			}
		}
	}()
}
