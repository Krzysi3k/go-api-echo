package handlers

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/labstack/echo/v4"
	"github.com/redis/go-redis/v9"
)

//go:embed docker_compose_path.txt
var composePath string

var validKeys = map[string]bool{
	"termometr-payload":      true,
	"termometr-payload-imgw": true,
	"termometr-wew-payload":  true,
	"air-quality-gios":       true,
	"power:usage:gniazdko-1": true,
	"power:usage:gniazdko-3": true,
	"power:usage:gniazdko-4": true,
}

func GetRedisData(ctx context.Context, rdb *redis.Client) echo.HandlerFunc {

	return func(e echo.Context) error {
		// ctx := context.Background()
		keyName := e.QueryParam("data")
		if keyName == "" {
			return e.JSON(400, map[string]string{"payload": "missing query string"})
		}
		if keyName == "job:offers" {
			offers := FetchOffers(ctx, rdb)
			fromAmount := e.QueryParam("gt")
			if fromAmount == "" {
				return e.JSON(200, offers)
			}
			from, err := strconv.Atoi(fromAmount)
			if err != nil {
				return e.JSON(400, map[string]string{"payload": "wrong query param"})
			}
			filteredOffers := make([]JobOffer, 0)
			for _, i := range offers {
				if i.Maxsalary >= from {
					filteredOffers = append(filteredOffers, i)
				}
			}
			return e.JSON(200, filteredOffers)
		}
		val, err := rdb.Get(ctx, keyName).Result()
		if err != nil {
			return e.JSON(404, map[string]string{"payload": "key not found"})
		}
		if strings.Contains(keyName, "docker:metrics:") || validKeys[keyName] {
			return e.String(200, val)
		}
		if json.Valid([]byte(val)) {
			return e.JSONBlob(200, []byte(val))
		} else {
			return e.JSON(200, map[string]string{"payload": val})
		}
	}
}

// @Summary Show most used keys from Redis.
// @Description shows vibration-sensor, door-state. humiditym and washing-state.
// @Tags redis
// @Accept */*
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /redis-info [get]
func GetRedisInfo(ctx context.Context, rdb *redis.Client) echo.HandlerFunc {

	return func(c echo.Context) error {
		// ctx := context.Background()
		keys := []string{
			"vibration-sensor",
			"door-state",
			"humidity",
			"washing-state",
		}
		val := rdb.MGet(ctx, keys...).Val()
		var sb strings.Builder
		sb.WriteString("{")
		for i := 0; i < len(keys); i++ {
			if val[i] != nil {
				if v, ok := val[i].(string); ok {
					if json.Valid([]byte(v)) {
						sb.WriteString(`"` + keys[i] + `":` + v + ",")
					} else {
						sb.WriteString(`"` + keys[i] + `":"` + v + `",`)
					}
				}
			}
		}
		redisKeys := rdb.Keys(ctx, "*").Val()
		sb.WriteString(fmt.Sprintf("\"Redis keys-in-use\":%v}", len(redisKeys)))
		return c.JSONBlob(200, []byte(sb.String()))
	}
}

// @Summary Show docker images or containers.
// @Description shows docker containers/images basic information.
// @Tags docker
// @Accept */*
// @Produce json
// @Param items query string false "item type: containers or images"
// @Success 200 {object} map[string]interface{}
// @Router /docker-info [get]
func GetDockerInfo(ctx context.Context, dockerClient *client.Client) echo.HandlerFunc {

	return func(c echo.Context) error {
		// ctx := context.Background()
		queryParam := c.QueryParam("items")
		switch queryParam {
		case "containers":
			containerList, err := dockerClient.ContainerList(ctx, types.ContainerListOptions{All: true})
			logError(err)
			containers := []string{}
			for _, container := range containerList {
				conName := strings.Replace(container.Names[0], "/", "", -1)
				line := fmt.Sprintf("%v - %v", conName, container.Status)
				containers = append(containers, line)
			}
			return c.JSON(200, map[string][]string{"containers": containers})
		case "images":
			imagesList, err := dockerClient.ImageList(ctx, types.ImageListOptions{All: true})
			logError(err)
			images := []string{}
			for _, img := range imagesList {
				if len(img.RepoTags) > 0 {
					line := fmt.Sprintf("%v, %vMB", img.RepoTags[0], (img.Size / 1024 / 1024))
					images = append(images, line)
				}
			}
			return c.JSON(200, map[string][]string{"images": images})
		default:
			return c.JSON(400, map[string]string{"payload": "wrong or missing query param"})
		}
	}
}

func RemoveContainer(ctx context.Context, dockerClient *client.Client) echo.HandlerFunc {

	return func(c echo.Context) error {
		// ctx := context.Background()
		nameParam := c.QueryParam("name")
		stopParam := c.QueryParam("stop")
		if nameParam == "" {
			return c.JSON(400, map[string]string{"payload": "missing query param"})
		}
		containerList, err := dockerClient.ContainerList(ctx, types.ContainerListOptions{All: true})
		if err != nil {
			log.Fatal(err)
		}
		for _, cont := range containerList {
			for _, cname := range cont.Names {
				if cname == "/"+nameParam {
					if stopParam == "true" {
						if err := dockerClient.ContainerRemove(ctx, cont.ID, types.ContainerRemoveOptions{Force: true}); err != nil {
							return c.JSON(200, map[string]string{"info": fmt.Sprint(err)})
						}
					} else {
						if err := dockerClient.ContainerRemove(ctx, cont.ID, types.ContainerRemoveOptions{}); err != nil {
							return c.JSON(200, map[string]string{"info": fmt.Sprint(err)})
						}
					}
					return c.JSON(200, map[string]string{"removed": nameParam})
				}
			}
		}
		return c.JSON(404, map[string]string{"not found": nameParam})
	}
}

// @Summary Show docker container logs.
// @Description shows docker logs of running container.
// @Tags docker
// @Accept */*
// @Produce plain
// @Param container query string false "container name"
// @Param since query int false "since when in minutes"
// @Success 200 {string} string
// @Router /docker-logs [get]
func GetContainerLogs(ctx context.Context, dockerClient *client.Client) echo.HandlerFunc {
	return func(c echo.Context) error {
		since, err := strconv.Atoi(c.QueryParam("since"))
		if err != nil {
			return c.JSON(400, map[string]string{"payload": "wrong query param type, required number"})
		}
		containerName := c.QueryParam("container")
		if containerName == "" {
			return c.JSON(400, map[string]string{"payload": "container query param is empty, required name"})
		}
		options := types.ContainerLogsOptions{
			ShowStdout: true,
			ShowStderr: true,
			Follow:     false,
			Since:      time.Now().Add(-time.Duration(since) * time.Minute).Format(time.RFC3339),
		}

		out, err := dockerClient.ContainerLogs(ctx, containerName, options)
		if err != nil {
			panic(err)
		}
		defer out.Close()

		var buf bytes.Buffer
		_, err = stdcopy.StdCopy(&buf, &buf, out)
		if err != nil {
			log.Fatal(err)
		}
		return c.String(200, buf.String())
	}
}

func UpContainerStack(dockerClient *client.Client) echo.HandlerFunc {
	return func(c echo.Context) error {
		exec.Command("sh", composePath+"/up.sh").Run()
		return c.JSON(200, map[string]string{"command": "docker-compose up"})
	}
}

func DeleteContainerMetrics(ctx context.Context, rdb *redis.Client) echo.HandlerFunc {
	return func(c echo.Context) error {
		// ctx := context.Background()
		metricKeys := rdb.Keys(ctx, "docker:metrics*").Val()
		rdb.Del(ctx, metricKeys...)
		return c.JSON(200, map[string][]string{"metrics-removed": metricKeys})
	}
}

func DeleteRedisKeys(ctx context.Context, rdb *redis.Client) echo.HandlerFunc {
	return func(c echo.Context) error {
		// ctx := context.Background()
		keyName := c.QueryParam("key")
		redisKeys := rdb.Keys(ctx, keyName).Val()
		rdb.Del(ctx, redisKeys...)
		return c.JSON(200, map[string][]string{"keys-removed": redisKeys})
	}
}

func logError(err error) {
	if err != nil {
		log.Println(err)
	}
}
