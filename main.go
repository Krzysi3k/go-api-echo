package main

import (
	"context"
	"log"

	"github.com/docker/docker/client"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/redis/go-redis/v9"

	"go-api-echo/pkg/handlers"
	"go-api-echo/pkg/metrics"

	_ "go-api-echo/docs"

	echoSwagger "github.com/swaggo/echo-swagger"
)

// @title Homestack API
// @version 1.0
// @description This is helper API for IoT and other things.
// @host api.wyselab.pl
// @BasePath /api/v1
// @schemes https
func main() {

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
	// e.Use(middleware.CORS())
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowHeaders: []string{"*"},
		// AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept,
		// 	"X-Requested-With", "Content-Type", "Authorization", "X-Api-Key", "Access-Control-Allow-Origin",
		// },
	}))
	// e.Use(middleware.SecureWithConfig(middleware.DefaultSecureConfig))

	e.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
		Format: `{"date":"${time_rfc3339}","ip":"${remote_ip}","method":"${method}","status":"${status}","response_time":"${latency_human}","uri":"${uri}","agent":"${user_agent}"}` + "\n",
	}))

	err = godotenv.Load("/home/krzysiek/go-api-echo/.env")
	if err != nil {
		log.Fatal("cannot load .env file")
	}

	ctx := context.Background()

	apiV1 := e.Group("/api/v1")
	apiV1.GET("/get-redis-data", handlers.GetRedisData(ctx, rdb))
	apiV1.GET("/redis-info", handlers.GetRedisInfo(ctx, rdb))
	apiV1.DELETE("/redis-keys", handlers.DeleteRedisKeys(ctx, rdb))
	apiV1.GET("/docker-info", handlers.GetDockerInfo(ctx, dockerClient))
	apiV1.GET("/docker-logs", handlers.GetContainerLogs(ctx, dockerClient))
	apiV1.GET("/containers-up", handlers.UpContainerStack(dockerClient))
	apiV1.DELETE("/container", handlers.RemoveContainer(ctx, dockerClient))
	apiV1.DELETE("/container-metrics", handlers.DeleteContainerMetrics(ctx, rdb))
	apiV1.GET("/random", metrics.RandomHandler(rdb))
	apiV1.GET("/metrics", echo.WrapHandler(promhttp.Handler()))
	apiV1.POST("/webhook", handlers.ProcessIncomingMessage())

	e.GET("/swagger/*", echoSwagger.WrapHandler)

	e.Logger.Fatal(e.Start(":5001"))
}
