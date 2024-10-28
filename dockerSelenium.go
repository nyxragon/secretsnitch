package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	docker "github.com/fsouza/go-dockerclient"
)

func scrapeWithSelenium(url string) string {
	client, err := docker.NewClient("unix:///var/run/docker.sock")
	if err != nil {
		fmt.Println("Error creating Docker client:", err)
		return ""
	}

	currentDir, err := os.Getwd()
	if err != nil {
		fmt.Println("Error getting current directory:", err)
		return ""
	}
	hostVolumePath := filepath.Join(currentDir, ".urlCache")
	containerVolumePath := "/app/.urlCache"

	containerOptions := docker.CreateContainerOptions{
		Config: &docker.Config{
			Image: "selenium-fetcher",
			Cmd:   []string{url},
		},
		HostConfig: &docker.HostConfig{
			Binds: []string{fmt.Sprintf("%s:%s", hostVolumePath, containerVolumePath)},
		},
	}

	container, err := client.CreateContainer(containerOptions)
	if err != nil {
		fmt.Println("Error creating container:", err)
		return ""
	}

	if err := client.StartContainer(container.ID, nil); err != nil {
		fmt.Println("Error starting container:", err)
		return ""
	}

	status, err := client.WaitContainer(container.ID)
	if err != nil {
		fmt.Println("Error waiting for container:", err)
		return ""
	}

	if err := client.RemoveContainer(docker.RemoveContainerOptions{ID: container.ID, Force: true}); err != nil {
		fmt.Println("Error removing container:", err)
	}

	if status == 0 {
		log.Printf("Content from %s saved to %s\n", url, cacheDir)
		return makeCacheFilename(url)
	}

	return ""

}
