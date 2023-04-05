package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/labstack/echo/v4"
	"github.com/redis/go-redis/v9"
)

//go:embed docker_compose_path.txt
var composePath string

func GetRedisData(ctx context.Context, rdb *redis.Client) echo.HandlerFunc {

	return func(e echo.Context) error {
		keyName := e.QueryParam("data")
		if keyName == "" {
			return e.JSON(400, map[string]interface{}{"payload": "missing query string"})
		}
		val, err := rdb.Get(ctx, keyName).Result()
		if err != nil {
			return e.JSON(404, map[string]interface{}{"payload": "key not found"})
		}
		if strings.Contains(keyName, "docker:metrics:") || keyName == "termometr-payload" {
			return e.String(200, val)
		}
		if json.Valid([]byte(val)) {
			return e.JSONBlob(200, []byte(val))
		} else {
			return e.JSON(200, map[string]interface{}{"payload": val})
		}
	}
}

func GetRedisInfo(ctx context.Context, rdb *redis.Client) echo.HandlerFunc {

	return func(c echo.Context) error {
		keys := []string{
			"vibration-sensor",
			"door-state",
			"rotate-option",
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

func GetDockerInfo(ctx context.Context, dockerClient *client.Client) echo.HandlerFunc {

	return func(c echo.Context) error {
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
			return c.JSON(200, map[string]interface{}{"containers": containers})
		case "images":
			imagesList, err := dockerClient.ImageList(ctx, types.ImageListOptions{All: true})
			logError(err)
			images := []string{}
			for _, img := range imagesList {
				if strings.Contains(img.RepoTags[0], "<none>") {
					continue
				}
				line := fmt.Sprintf("%v, %vMB", img.RepoTags[0], (img.Size / 1024 / 1024))
				images = append(images, line)
			}
			return c.JSON(200, map[string]interface{}{"images": images})
		default:
			return c.JSON(400, map[string]interface{}{"payload": "wrong or missing query param"})
		}
	}
}

func RemoveContainer(ctx context.Context, dockerClient *client.Client) echo.HandlerFunc {

	return func(c echo.Context) error {
		nameParam := c.QueryParam("name")
		stopParam := c.QueryParam("stop")
		if nameParam == "" {
			return c.JSON(400, map[string]interface{}{"payload": "missing query param"})
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
							return c.JSON(200, map[string]interface{}{"info": fmt.Sprint(err)})
						}
					} else {
						if err := dockerClient.ContainerRemove(ctx, cont.ID, types.ContainerRemoveOptions{}); err != nil {
							return c.JSON(200, map[string]interface{}{"info": fmt.Sprint(err)})
						}
					}
					return c.JSON(200, map[string]interface{}{"removed": nameParam})
				}
			}
		}
		return c.JSON(404, map[string]interface{}{"not found": nameParam})
	}
}

func UpContainerStack(ctx context.Context, dockerClient *client.Client) echo.HandlerFunc {
	return func(c echo.Context) error {
		exec.Command("sh", composePath+"/up.sh").Run()
		return c.JSON(200, map[string]interface{}{"command": "docker-compose up"})
	}
}

func logError(err error) {
	if err != nil {
		log.Println(err)
	}
}
