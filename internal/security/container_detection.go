package security

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ContainerEnvironment represents the detected container runtime
type ContainerEnvironment string

const (
	// ContainerDocker indicates Docker container
	ContainerDocker ContainerEnvironment = "docker"
	// ContainerKubernetes indicates Kubernetes pod
	ContainerKubernetes ContainerEnvironment = "kubernetes"
	// ContainerPodman indicates Podman container
	ContainerPodman ContainerEnvironment = "podman"
	// ContainerNone indicates not running in a container
	ContainerNone ContainerEnvironment = "none"
)

// IsRunningInContainer detects if the process is running inside a container
// Returns the container environment type and a boolean indicating detection
func IsRunningInContainer() (ContainerEnvironment, bool) {
	// Check for Kubernetes first (most specific)
	if isKubernetes() {
		return ContainerKubernetes, true
	}

	// Check for Docker
	if isDocker() {
		return ContainerDocker, true
	}

	// Check for Podman
	if isPodman() {
		return ContainerPodman, true
	}

	return ContainerNone, false
}

// isKubernetes checks for Kubernetes-specific indicators
func isKubernetes() bool {
	// Check for Kubernetes service account token
	if _, err := os.Stat("/var/run/secrets/kubernetes.io/serviceaccount/token"); err == nil {
		return true
	}

	// Check for Kubernetes environment variables
	if os.Getenv("KUBERNETES_SERVICE_HOST") != "" {
		return true
	}

	if os.Getenv("KUBERNETES_PORT") != "" {
		return true
	}

	return false
}

// isDocker checks for Docker-specific indicators
func isDocker() bool {
	// Check for .dockerenv file (traditional Docker indicator)
	if _, err := os.Stat("/.dockerenv"); err == nil {
		return true
	}

	// Check cgroup for docker
	if hasCgroupIndicator("docker") {
		return true
	}

	return false
}

// isPodman checks for Podman-specific indicators
func isPodman() bool {
	// Podman sets specific environment variables
	if os.Getenv("container") == "podman" {
		return true
	}

	// Check cgroup for podman
	if hasCgroupIndicator("podman") {
		return true
	}

	return false
}

// hasCgroupIndicator checks if cgroup file contains the specified container runtime
func hasCgroupIndicator(runtime string) bool {
	// Read /proc/self/cgroup
	data, err := os.ReadFile("/proc/self/cgroup")
	if err != nil {
		return false
	}

	content := string(data)
	return strings.Contains(content, runtime)
}

// IsEphemeralPath checks if a file path is in ephemeral container storage
// Returns true if the path is unsafe for persistent data in containers
func IsEphemeralPath(path string) bool {
	// Get absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		// If we can't resolve path, treat it as potentially ephemeral for safety
		absPath = path
	}

	// Common ephemeral paths in containers
	ephemeralPrefixes := []string{
		"/tmp",
		"/var/tmp",
		"/run",
		"/var/run",
		"/dev/shm",
	}

	for _, prefix := range ephemeralPrefixes {
		if strings.HasPrefix(absPath, prefix) {
			return true
		}
	}

	// Check if path is in a Kubernetes emptyDir volume
	// emptyDir volumes are typically mounted under /var/lib/kubelet/pods/
	// but can be anywhere. We check for common patterns.
	if strings.Contains(absPath, "/emptyDir") {
		return true
	}

	return false
}

// ValidatePathForPersistence checks if a file path is safe for persistent storage
// in the current environment. Returns an error with remediation guidance if unsafe.
func ValidatePathForPersistence(path string, purpose string) error {
	env, inContainer := IsRunningInContainer()

	// If not in a container, all paths are safe (host filesystem)
	if !inContainer {
		return nil
	}

	// In containers, check for ephemeral paths
	if IsEphemeralPath(path) {
		return fmt.Errorf(
			"unsafe persistent storage: %s path '%s' is in ephemeral storage in %s container - "+
				"this path will be WIPED on container restart, causing loss of all %s data, "+
				"security vulnerabilities (replay attacks possible after restart), and data corruption on pod rescheduling - "+
				"use distributed storage (Redis, PostgreSQL), mount a persistent volume (PVC), or use /data directory with persistent volume "+
				"(see REPLAY_STORE_MIGRATION_GUIDE.md for complete instructions)",
			purpose, path, env, purpose)
	}

	// Path appears safe for persistent storage in container
	return nil
}

// GetContainerInfo returns a human-readable description of the container environment
func GetContainerInfo() string {
	env, inContainer := IsRunningInContainer()
	if !inContainer {
		return "Not running in a container (bare metal/VM)"
	}

	return fmt.Sprintf("Running in %s container", env)
}

// ShouldEnforceContainerSafety determines if container safety checks should be enforced
// Returns true if running in a container environment where ephemeral storage is risky
func ShouldEnforceContainerSafety() bool {
	_, inContainer := IsRunningInContainer()
	return inContainer
}
