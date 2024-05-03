//go:build integration
// +build integration

package main_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/GabrieldeFreire/rate-limit/cache"
	"github.com/GabrieldeFreire/rate-limit/rate"
	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
	redisContainer "github.com/testcontainers/testcontainers-go/modules/redis"
)

type TestSuite struct {
	suite.Suite
	api            *http.ServeMux
	cacheEndpoint  string
	cacheContainer *redisContainer.RedisContainer
	ctx            context.Context
}

func (suite *TestSuite) SetupTest() {
	suite.ctx = context.Background()
	redisContainer, err := redisContainer.RunContainer(suite.ctx, testcontainers.WithImage("redis:latest"))
	suite.cacheContainer = redisContainer
	require.NoError(suite.T(), err)
	suite.cacheEndpoint, err = redisContainer.Endpoint(suite.ctx, "")
	require.NoError(suite.T(), err)

	os.Setenv("MAX_REQUESTS", "10")
	os.Setenv("WINDOW_SECONDS", "10")
	os.Setenv("BLOCK_SECONDS", "10")
}

func (suite *TestSuite) TearDownTest() {
	err := suite.cacheContainer.Terminate(suite.ctx)
	require.NoError(suite.T(), err)
}

func (suite *TestSuite) setupApi() {
	opt := &redis.Options{
		Addr:     suite.cacheEndpoint,
		Password: "",
		DB:       0,
	}

	cacheStrategy := &cache.RedisStrategy{
		Client: redis.NewClient(opt),
		Ctx:    context.Background(),
	}

	handler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "Welcome home!")
	}
	rateMiddleware := rate.CreateLimitRateMiddleware(cacheStrategy)
	suite.api = http.NewServeMux()
	suite.api.HandleFunc("GET /", rateMiddleware(http.HandlerFunc(handler)))
}

func (suite *TestSuite) TestGetEndpointShouldAssertReturnMessage() {
	suite.setupApi()
	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		suite.T().Fatal(err)
	}

	rr := httptest.NewRecorder()
	suite.api.ServeHTTP(rr, req)
	suite.Assert().Equal(http.StatusOK, rr.Code)

	expected := "Welcome home!"
	body, err := io.ReadAll(rr.Body)
	suite.Assert().NoError(err)
	suite.Assert().Equal(expected, string(body))
}

func (suite *TestSuite) TestGetEndpointShouldBlockRequestByIp() {
	maxRequests := 10
	windowSeconds := 2
	blockSeconds := 3
	os.Setenv("MAX_REQUESTS", fmt.Sprintf("%d", maxRequests))
	os.Setenv("WINDOW_SECONDS", fmt.Sprintf("%d", windowSeconds))
	os.Setenv("BLOCK_SECONDS", fmt.Sprintf("%d", blockSeconds))

	suite.setupApi()
	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		suite.T().Fatal(err)
	}

	for i := 0; i < maxRequests+1; i++ {
		time.Sleep(1 * time.Millisecond)
		rr := httptest.NewRecorder()

		suite.api.ServeHTTP(rr, req)
		if i < maxRequests {
			suite.Assert().Equal(http.StatusOK, rr.Code)
		} else {
			suite.Assert().Equal(http.StatusTooManyRequests, rr.Code)
		}
	}
}

func (suite *TestSuite) TestGetEndpointShouldBlockRequestByToken() {
	maxRequests := 10
	windowSeconds := 2
	blockSeconds := 3
	os.Setenv("MAX_REQUESTS", fmt.Sprintf("%d", maxRequests))
	os.Setenv("WINDOW_SECONDS", fmt.Sprintf("%d", windowSeconds))
	os.Setenv("BLOCK_SECONDS", fmt.Sprintf("%d", blockSeconds))

	suite.setupApi()

	tokens := []string{"SomeToken", "AnotherToken"}
	for _, token := range tokens {
		req, err := http.NewRequest("GET", "/", nil)
		req.Header.Set("API_KEY", token)
		if err != nil {
			suite.T().Fatal(err)
		}
		for i := 0; i < maxRequests+1; i++ {
			time.Sleep(time.Millisecond)
			rr := httptest.NewRecorder()

			suite.api.ServeHTTP(rr, req)
			if i < maxRequests {
				suite.Assert().Equal(http.StatusOK, rr.Code)
			} else {
				suite.Assert().Equal(http.StatusTooManyRequests, rr.Code)
			}
		}
	}
}

func (suite *TestSuite) TestGetEndpointShouldAllowAfterBlockTime() {
	maxRequests := 10
	windowSeconds := 2
	blockSeconds := 3
	os.Setenv("MAX_REQUESTS", fmt.Sprintf("%d", maxRequests))
	os.Setenv("WINDOW_SECONDS", fmt.Sprintf("%d", windowSeconds))
	os.Setenv("BLOCK_SECONDS", fmt.Sprintf("%d", blockSeconds))

	suite.setupApi()
	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		suite.T().Fatal(err)
	}

	for i := 0; i < maxRequests+1; i++ {
		time.Sleep(time.Millisecond)
		rr := httptest.NewRecorder()

		suite.api.ServeHTTP(rr, req)
		if i < maxRequests {
			suite.Assert().Equal(http.StatusOK, rr.Code)
		} else {
			suite.Assert().Equal(http.StatusTooManyRequests, rr.Code)
		}
	}
	var endBreak time.Time
	startBreak := time.Now()
	for {
		rr := httptest.NewRecorder()
		suite.api.ServeHTTP(rr, req)
		if rr.Code == http.StatusOK {
			endBreak = time.Now()
			break
		}
	}
	delta := endBreak.Sub(startBreak).Seconds()
	precision := 0.001
	suite.Assert().GreaterOrEqual(delta, float64(blockSeconds)-precision)
	suite.Assert().LessOrEqual(delta, float64(blockSeconds)+precision)
}

func TestRateLimitSuite(t *testing.T) {
	suite.Run(t, new(TestSuite))
}
