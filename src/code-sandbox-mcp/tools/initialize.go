package tools

import (
	"context"
	"fmt"
	"log"
	"io"
	// "os"

	dockerImage "github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/mark3labs/mcp-go/mcp"
)

// InitializeEnvironment creates a new container for code execution
func InitializeEnvironment(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Get the requested Docker image or use default
	image, ok := request.Params.Arguments["image"].(string)
	if !ok || image == "" {
		// Default to a slim debian image with Python pre-installed
		image = "python:3.12-slim-bookworm"
	}

	name, ok := request.Params.Arguments["name"].(string)
	if !ok {
		// Default to a slim debian image with Python pre-installed
		name = ""
	}

	// Create and start the container
	containerId, err := createContainer(ctx, image, name)
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("Error: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("container_id: %s", containerId)), nil
}

// createContainer creates a new Docker container and returns its ID
func createContainer(ctx context.Context, image string, name string) (string, error) {
	cli, err := client.NewClientWithOpts(
		client.FromEnv,
		client.WithAPIVersionNegotiation(),
	)
	if err != nil {
		return "", fmt.Errorf("failed to create Docker client: %w", err)
	}
	defer cli.Close()

	// check whether exist
	if name!="" {
		log.Printf("Finding container with existing name: %s",name)
		info, err := cli.ContainerInspect(ctx, name)
		if err!=nil && !client.IsErrNotFound(err){
			return "",fmt.Errorf("failed to check existing Docker container: %w", err)
		}

		if err == nil {
			if info.State != nil && info.State.Running {
				log.Printf("Found existing container %s",info.ID)
				return info.ID,nil
			}
			// else if container not running, rm it
			if err := cli.ContainerRemove(ctx, name, container.RemoveOptions{}); err != nil {
				return "",fmt.Errorf("failed to rm existing stopped Docker container: %w", err)
			}
		}
	}

	// Pull the Docker image if not already available
	reader, err := cli.ImagePull(ctx, image, dockerImage.PullOptions{})
	// log.Printf("Pulling image %s",image)
	if err != nil {
		return "", fmt.Errorf("failed to pull Docker image %s: %w", image, err)
	}
	defer reader.Close()
	// io.Copy(os.Stdout, reader)
	io.Copy(io.Discard, reader)
	// log.Printf("Image %s ready",image)

	// Create container config with a working directory
	config := &container.Config{
		Image:      image,
		WorkingDir: "/app",
		Tty:        true,
		OpenStdin:  true,
		StdinOnce:  false,
	}

	// Create host config
	hostConfig := &container.HostConfig{
		// Add any resource constraints here if needed
	}

	// Create the container
	resp, err := cli.ContainerCreate(
		ctx,
		config,
		hostConfig,
		nil,
		nil,
		name,
	)
	if err != nil {
		return "", fmt.Errorf("failed to create container: %w", err)
	}

	// Start the container
	if err := cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return "", fmt.Errorf("failed to start container: %w", err)
	}

	log.Printf("Container Ready %s:%s",name,resp.ID)
	return resp.ID, nil
}
