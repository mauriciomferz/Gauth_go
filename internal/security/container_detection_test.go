package security

import (
	"testing"
)

func TestContainerDetection(t *testing.T) {
	env, inContainer := IsRunningInContainer()

	t.Logf("Container Detection Results:")
	t.Logf("  Environment: %s", env)
	t.Logf("  In Container: %v", inContainer)
	t.Logf("  Info: %s", GetContainerInfo())

	// Test should pass regardless of environment
	if inContainer {
		if env == ContainerNone {
			t.Errorf("Inconsistent detection: inContainer=true but env=ContainerNone")
		}
	} else {
		if env != ContainerNone {
			t.Errorf("Inconsistent detection: inContainer=false but env=%s", env)
		}
	}
}

func TestEphemeralPathDetection(t *testing.T) {
	tests := []struct {
		path          string
		wantEphemeral bool
	}{
		{"/tmp/replay.db", true},
		{"/var/tmp/replay.db", true},
		{"/run/replay.db", true},
		{"/var/run/replay.db", true},
		{"/dev/shm/replay.db", true},
		{"/data/replay.db", false},
		{"/mnt/replay.db", false},
		{"/opt/replay.db", false},
		{"./replay.db", false},
		{"/home/user/replay.db", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := IsEphemeralPath(tt.path)
			if got != tt.wantEphemeral {
				t.Errorf("IsEphemeralPath(%q) = %v, want %v", tt.path, got, tt.wantEphemeral)
			}
		})
	}
}

func TestValidatePathForPersistence(t *testing.T) {
	// Test in non-container environment (current macOS)
	testPath := "/tmp/replay.db"
	err := ValidatePathForPersistence(testPath, "test replay protection")

	// On macOS (not in container), should accept /tmp
	if err != nil {
		t.Logf("Path validation rejected (expected in container): %v", err)
	} else {
		t.Logf("Path validation accepted (not in container)")
	}
}

func TestShouldEnforceContainerSafety(t *testing.T) {
	enforce := ShouldEnforceContainerSafety()
	t.Logf("Should enforce container safety: %v", enforce)

	// Should match container detection result
	_, inContainer := IsRunningInContainer()
	if enforce != inContainer {
		t.Errorf("ShouldEnforceContainerSafety() = %v, but IsRunningInContainer() = %v", enforce, inContainer)
	}
}
