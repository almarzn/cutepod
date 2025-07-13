package e2e

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func TestCuteE2E(t *testing.T) {
	cuteBin := "../bin/cutepod"
	imageTag := "localhost/e2e-server:test"
	containerName := "e2e-test-container"

	rm := exec.Command("podman", "rm", "-f", "e2e-demo-chart-container")
	rm.Stdout, rm.Stderr = os.Stdout, os.Stderr
	if err := rm.Run(); err != nil {
		t.Fatalf("failed to remove container: %v", err)
	}

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working dir: %v", err)
	}

	// Step 2: Build the container
	t.Log("Building test container...")
	build := exec.Command("podman", "build", "-t", imageTag, "--network=none", "-f", "Containerfile", ".")
	build.Dir = cwd
	build.Stdout, build.Stderr = os.Stdout, os.Stderr
	if err := build.Run(); err != nil {
		t.Fatalf("failed to build container: %v", err)
	}

	// Step 3: Run `cute install`
	t.Log("Running cutepod install...")
	install := exec.Command(cuteBin, "install", "e2e", cwd+"/chart")
	install.Stdout, install.Stderr = os.Stdout, os.Stderr
	if err := install.Run(); err != nil {
		t.Fatalf("cute install failed: %v", err)
	}

	// Step 5: Wait for the container to become ready
	url := "http://localhost:18080"
	t.Logf("Waiting for container at %s...", url)
	if err := waitForReady(url, 10*time.Second); err != nil {
		t.Fatalf("container did not respond: %v", err)
	}

	// Step 6: Verify the HTTP response
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("GET failed: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if !strings.Contains(string(body), "Hello, world!") {
		t.Fatalf("unexpected response: %q", string(body))
	}
	t.Log("Container response verified!")

	// Step 7: Cleanup
	t.Log("Cleaning up...")
	exec.Command("podman", "rm", "-f", containerName).Run()
	exec.Command("podman", "rmi", "-f", imageTag).Run()
}

func waitForReady(url string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for {
		resp, err := http.Get(url)
		if err == nil && resp.StatusCode == http.StatusOK {
			resp.Body.Close()
			return nil
		}
		if time.Now().After(deadline) {
			if err != nil {
				return fmt.Errorf("timeout: %v", err)
			}
			return fmt.Errorf("timeout: got status %v", resp.Status)
		}
		time.Sleep(300 * time.Millisecond)
	}
}
