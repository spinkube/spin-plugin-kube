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
			},
			expected: "scaffold_image.yml",
		},
		{
			name: "runtime config is provided",
			opts: ScaffoldOptions{
				from:       "ghcr.io/foo/example-app:v0.1.0",
				replicas:   2,
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
