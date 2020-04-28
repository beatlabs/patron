package http

import (
	"context"
	"net/http"
	"testing"

	"github.com/beatlabs/patron/cache"
	"github.com/stretchr/testify/assert"
)

type builderOperation func(routeBuilder *RouteBuilder) *RouteBuilder

type cacheBuilderOperation func(routeBuilder *RouteCache) *RouteCache

type arg struct {
	bop  builderOperation
	cbop cacheBuilderOperation
	age  Age
	err  bool
}

func TestNewRouteCacheBuilder(t *testing.T) {

	args := []arg{
		{
			bop: func(routeBuilder *RouteBuilder) *RouteBuilder {
				return routeBuilder.MethodGet()
			},
			age: Age{Max: 10},
		},
		// error with '0' ttl
		{
			bop: func(routeBuilder *RouteBuilder) *RouteBuilder {
				return routeBuilder.MethodGet()
			},
			age: Age{},
			err: true,
		},
		// error for POST method
		{
			bop: func(routeBuilder *RouteBuilder) *RouteBuilder {
				return routeBuilder.MethodPost()
			},
			age: Age{Max: 10},
			err: true,
		}, // error for instant function nil
		{
			bop: func(routeBuilder *RouteBuilder) *RouteBuilder {
				return routeBuilder.MethodGet()
			},
			cbop: func(routeBuilder *RouteCache) *RouteCache {
				routeBuilder.WithTimeInstant(nil)
				return routeBuilder
			},
			age: Age{Max: 10},
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

	routeCacheConfig := NewRouteCache(cache, arg.age)

	if arg.cbop != nil {
		routeCacheConfig = arg.cbop(routeCacheConfig)
	}

	routeBuilder.WithRouteCache(routeCacheConfig)

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
