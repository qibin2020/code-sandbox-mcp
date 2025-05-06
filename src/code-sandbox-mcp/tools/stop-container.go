package tools

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/mark3labs/mcp-go/mcp"
)

// StopContainer stops and removes a container by its ID
func StopContainer(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Get the container ID from the request
	containerId, ok := request.Params.Arguments["container_id"].(string)
	if !ok || containerId == "" {
		return mcp.NewToolResultText("Error: container_id is required"), nil
	}

	// Stop and remove the container
	if err := stopAndRemoveContainer(ctx, containerId); err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("Error: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Successfully stopped and removed container: %s", containerId)), nil
}

// stopAndRemoveContainer stops and removes a Docker container
func stopAndRemoveContainer(ctx context.Context, containerId string) error {
	cli, err := client.NewClientWithOpts(
		client.FromEnv,
		client.WithAPIVersionNegotiation(),
	)
	if err != nil {
		return fmt.Errorf("failed to create Docker client: %w", err)
	}
	defer cli.Close()

	// Stop the container with a timeout
	// Now we do it forcely (since exec is fg)
	timeout := 10 // seconds
	if err := cli.ContainerStop(ctx, containerId, container.StopOptions{Timeout: &timeout, Signal: "SIGKILL"}); err != nil {
		return fmt.Errorf("failed to stop container: %w", err)
	}

	// Remove the container
	if err := cli.ContainerRemove(ctx, containerId, container.RemoveOptions{
		RemoveVolumes: true,
		Force:         true,
	}); err != nil {
		return fmt.Errorf("failed to remove container: %w", err)
	}

	return nil
}
