package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/joho/godotenv"
)

// Function to monitor logs of a specific container
func monitorContainerLogs(cli *client.Client, containerID string, criteria string) {
	ctx := context.Background()

	// Fetch container logs continuously
	reader, err := cli.ContainerLogs(ctx, containerID, container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     true, // Follow logs continuously
		Timestamps: false,
		Tail:       "1", // Get only the last log line
	})
	if err != nil {
		log.Fatalf("Error fetching logs for container %s: %v", containerID, err)
	}
	defer reader.Close()

	// Create a scanner to read the log lines
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		logLine := scanner.Text()

		// Check if the log line contains the criteria (e.g., "error")
		if strings.Contains(logLine, criteria) {
			fmt.Printf("Criteria '%s' found! Restarting container %s...\n", criteria, containerID)
			restartContainer(cli, containerID)
		}
	}

	if err := scanner.Err(); err != nil {
		log.Printf("Error reading logs for container %s: %v", containerID, err)
	}
}

// Function to restart the container
func restartContainer(cli *client.Client, containerID string) {
	ctx := context.Background()

	// Stop the container
	if err := cli.ContainerStop(ctx, containerID, container.StopOptions{}); err != nil {
		log.Printf("Error stopping container %s: %v", containerID, err)
		return
	}

	// Wait for a few seconds to ensure the container is stopped
	time.Sleep(10 * time.Second)

	// Start the container again
	if err := cli.ContainerStart(ctx, containerID, container.StartOptions{}); err != nil {
		log.Printf("Error starting container %s: %v", containerID, err)
	} else {
		fmt.Printf("Container %s restarted successfully.\n", containerID)
	}
}

// Function to resolve container name to container ID
func resolveContainerID(cli *client.Client, containerName string) (string, error) {
	ctx := context.Background()
	containerJSON, err := cli.ContainerInspect(ctx, containerName)
	if err != nil {
		return "", fmt.Errorf("container %s not found or not running", containerName)
	}
	return containerJSON.ID, nil
}

func main() {
	// Load the .env file
	err := godotenv.Load()
	if err != nil {
		fmt.Println("Error loading .env file, use system environemnt instead")
	}

	// Initialize Docker client
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Fatalf("Error initializing Docker client: %v", err)
	}

	// Fetch container name from environment variable
	containerName := os.Getenv("CONTAINER_NAME")
	if containerName == "" {
		log.Fatal("Environment variable CONTAINER_NAME not set.")
	}

	// Resolve the container name to container ID
	containerID, err := resolveContainerID(cli, containerName)
	if err != nil {
		log.Fatalf("Error resolving container name: %v", err)
	}

	// Define the log criteria (e.g., "error")
	msgErrCriteria := os.Getenv("ERROR_MSG")
	if msgErrCriteria == "" {
		log.Fatal("Environment variable ERROR_MSG not set.")
	}

	// Setup signal catching for graceful shutdown
	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, syscall.SIGINT, syscall.SIGTERM)

	// Run log monitoring in a separate goroutine
	go func() {
		for {
			monitorContainerLogs(cli, containerID, msgErrCriteria)
			// If monitoring stops, wait a bit and restart monitoring
			time.Sleep(2 * time.Second)
		}
	}()

	// Wait for termination signal (SIGINT or SIGTERM)
	<-stopChan
	fmt.Println("Shutting down gracefully...")
}
