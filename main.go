package main

import (
	"context"
	"log"

	"github.com/docker/docker/client"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	// "github.com/go-redis/redis/v9"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/redis/go-redis/v9"
)

func main() {

	var ctx = context.Background()
	rdb := redis.NewClient(&redis.Options{
		Addr:     "192.168.0.123:6379",
		Password: "",
		DB:       0,
	})

	dockerClient, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		log.Panic(err)
	}

	e := echo.New()
	e.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
		Format: `{"date":"${time_rfc3339}","ip":"${remote_ip}","method":"${method}","status":"${status}","response_time":"${latency_human}","uri":"${uri}","agent":"${user_agent}"}` + "\n",
	}))

	e.GET("/get-redis-data", GetRedisData(ctx, rdb))
	e.GET("/redis-info", GetRedisInfo(ctx, rdb))
	e.GET("/docker-info", GetDockerInfo(ctx, dockerClient))
	e.GET("/random", randomHandler(ctx, rdb))
	e.GET("/metrics", echo.WrapHandler(promhttp.Handler()))

	e.Logger.Fatal(e.Start(":5001"))
}
