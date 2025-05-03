package resources

import (
	"context"
	"fmt"
	"strings"
	"io"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/pkg/stdcopy"

	"github.com/docker/docker/client"
	"github.com/mark3labs/mcp-go/mcp"
)

func GetContainerLogs(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}
	defer cli.Close()

	containerIDPath, found := strings.CutPrefix(request.Params.URI, "containers://") // Extract ID from the full URI
	if !found {
		return nil, fmt.Errorf("invalid URI: %s", request.Params.URI)
	}
	containerID := strings.TrimSuffix(containerIDPath, "/logs")

	// Set default ContainerLogsOptions
	logOpts := container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
	}

	// Actually fetch the logs
	reader, err := cli.ContainerLogs(ctx, containerID, logOpts)
	if err != nil {
		return nil, fmt.Errorf("error fetching container logs: %w", err)
	}
	defer reader.Close()

	inspect, err := cli.ContainerInspect(ctx, containerID)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect container: %w", err)
	}

	var b strings.Builder
	if inspect.Config.Tty {
		// TTY containers output a raw stream
		if _, err := io.Copy(&b, reader); err != nil {
			return nil, fmt.Errorf("error reading TTY container logs: %w", err)
		}
	} else {
		// Non-TTY containers use multiplexed streams
		if _, err := stdcopy.StdCopy(&b, &b, reader); err != nil {
			return nil, fmt.Errorf("error copying container logs: %w", err)
		}
	}

	// Combine them. You could also return them separately if you prefer.
	combined := b.String()

	return []mcp.ResourceContents{
		mcp.TextResourceContents{
			URI:      fmt.Sprintf("containers://%s/logs", containerID),
			MIMEType: "text/plain",
			Text:     combined,
		},
	}, nil
}
