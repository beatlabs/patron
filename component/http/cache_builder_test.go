package http

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/beatlabs/patron/cache"
	"github.com/stretchr/testify/assert"
)

type builderOperation func(routeBuilder *RouteBuilder) *RouteBuilder

type cacheBuilderOperation func(routeBuilder *RouteCacheBuilder) *RouteCacheBuilder

type arg struct {
	bop  builderOperation
	cbop cacheBuilderOperation
	ttl  time.Duration
	err  bool
}

func TestNewRouteCacheBuilder(t *testing.T) {

	args := []arg{
		{
			bop: func(routeBuilder *RouteBuilder) *RouteBuilder {
				return routeBuilder.MethodGet()
			},
			ttl: 10,
		},
		// error with '0' ttl
		{
			bop: func(routeBuilder *RouteBuilder) *RouteBuilder {
				return routeBuilder.MethodGet()
			},
			ttl: 0,
			err: true,
		},
		// error for POST method
		{
			bop: func(routeBuilder *RouteBuilder) *RouteBuilder {
				return routeBuilder.MethodPost()
			},
			ttl: 10,
			err: true,
		},
		// error for maxFresh greater than ttl
		{
			bop: func(routeBuilder *RouteBuilder) *RouteBuilder {
				return routeBuilder.MethodGet()
			},
			cbop: func(routeBuilder *RouteCacheBuilder) *RouteCacheBuilder {
				routeBuilder.WithMaxFresh(10 + 1)
				return routeBuilder
			},
			ttl: 10,
			err: true,
		},
		// error for minAge value of '0'
		{
			bop: func(routeBuilder *RouteBuilder) *RouteBuilder {
				return routeBuilder.MethodGet()
			},
			cbop: func(routeBuilder *RouteCacheBuilder) *RouteCacheBuilder {
				routeBuilder.WithMinAge(0)
				return routeBuilder
			},
			ttl: 10,
			err: true,
		}, // error for instant function nil

		{
			bop: func(routeBuilder *RouteBuilder) *RouteBuilder {
				return routeBuilder.MethodGet()
			},
			cbop: func(routeBuilder *RouteCacheBuilder) *RouteCacheBuilder {
				routeBuilder.WithTimeInstant(nil)
				return routeBuilder
			},
			ttl: 10,
			err: true,
		},
	}

	c := newTestingCache()

	processor := func(context context.Context, request *Request) (response *Response, e error) {
		return nil, nil
	}

	handler := func(writer http.ResponseWriter, i *http.Request) {
	}

	for _, arg := range args {

		assertCacheBuilderBuild(t, arg, NewRouteBuilder("/", processor), c)

		assertCacheBuilderBuild(t, arg, NewRawRouteBuilder("/", handler), c)

	}
}

func assertCacheBuilderBuild(t *testing.T, arg arg, routeBuilder *RouteBuilder, cache cache.Cache) {

	routeCacheBuilder := NewRouteCacheBuilder(cache, arg.ttl)

	if arg.cbop != nil {
		routeCacheBuilder = arg.cbop(routeCacheBuilder)
	}

	routeBuilder.WithRouteCachedBuilder(routeCacheBuilder)

	if arg.bop != nil {
		routeBuilder = arg.bop(routeBuilder)
	}

	route, err := routeBuilder.Build()
	assert.NotNil(t, route)

	if arg.err {
		assert.Error(t, err)
	} else {
		assert.NoError(t, err)
	}
}
