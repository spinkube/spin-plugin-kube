package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestScaffoldCmd(t *testing.T) {
	testcases := []struct {
		name     string
		opts     ScaffoldOptions
		expected string
	}{
		{
			name: "only image is provided",
			opts: ScaffoldOptions{
				from:     "ghcr.io/foo/example-app:v0.1.0",
				replicas: 2,
				executor: "containerd-shim-spin",
			},
			expected: "scaffold_image.yml",
		},
		{
			name: "runtime config is provided",
			opts: ScaffoldOptions{
				from:       "ghcr.io/foo/example-app:v0.1.0",
				replicas:   2,
				executor:   "containerd-shim-spin",
				configfile: "testdata/runtime-config.toml",
			},
			expected: "scaffold_runtime_config.yml",
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			output, err := scaffold(tc.opts)
			require.Nil(t, err)

			expectedContent, err := os.ReadFile(filepath.Join("testdata", tc.expected))
			require.Nil(t, err)

			require.Equal(t, string(expectedContent), string(output))
		})
	}
}

func TestValidateImageReference_ValidImageReference(t *testing.T) {
	testCases := []string{
		"bacongobbler/hello-rust",
		"bacongobbler/hello-rust:v1.0.0",
		"ghcr.io/bacongobbler/hello-rust",
		"ghcr.io/bacongobbler/hello-rust:v1.0.0",
		"ghcr.io/spinkube/spinkube/runtime-class-manager:v1",
		"nginx:latest",
		"nginx",
	}

	for _, tc := range testCases {
		t.Run(tc, func(t *testing.T) {
			valid := validateImageReference(tc)
			require.True(t, valid, "Expected image reference to be valid")
		})

	}
}
