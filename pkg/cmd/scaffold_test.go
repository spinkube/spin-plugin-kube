package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestScaffoldOutput(t *testing.T) {
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
		{
			name: "one image pull secret is provided",
			opts: ScaffoldOptions{
				from:             "ghcr.io/foo/example-app:v0.1.0",
				replicas:         2,
				executor:         "containerd-shim-spin",
				configfile:       "testdata/runtime-config.toml",
				imagePullSecrets: []string{"secret-name"},
			},
			expected: "one_image_secret.yml",
		},
		{
			name: "multiple image pull secrets are provided",
			opts: ScaffoldOptions{
				from:             "ghcr.io/foo/example-app:v0.1.0",
				replicas:         2,
				executor:         "containerd-shim-spin",
				configfile:       "testdata/runtime-config.toml",
				imagePullSecrets: []string{"secret-name", "secret-name-2"},
			},
			expected: "multiple_image_secrets.yml",
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

func TestFlagValidation(t *testing.T) {
	testcases := []struct {
		name          string
		opts          ScaffoldOptions
		expectedError string
	}{
		{
			name: "valid HPA autoscaling options",
			opts: ScaffoldOptions{
				from:                              "ghcr.io/foo/example-app:v0.1.0",
				replicas:                          2,
				maxReplicas:                       5,
				executor:                          "containerd-shim-spin",
				autoscaler:                        "hpa",
				cpuLimit:                          "50m",
				memoryLimit:                       "100Mi",
				targetCpuUtilizationPercentage:    1,
				targetMemoryUtilizationPercentage: 1,
			},
		},
		{
			name: "valid KEDA autoscaling options",
			opts: ScaffoldOptions{
				from:                              "ghcr.io/foo/example-app:v0.1.0",
				replicas:                          2,
				maxReplicas:                       5,
				executor:                          "containerd-shim-spin",
				autoscaler:                        "keda",
				cpuLimit:                          "50m",
				memoryLimit:                       "100Mi",
				targetCpuUtilizationPercentage:    1,
				targetMemoryUtilizationPercentage: 1,
			},
		},
		{
			name: "invalid replica count",
			opts: ScaffoldOptions{
				from:     "ghcr.io/foo/example-app:v0.1.0",
				replicas: -1,
				executor: "containerd-shim-spin",
			},
			expectedError: "the minimum replica count (-1) must be greater than 0",
		},
		{
			name: "invalid image reference",
			opts: ScaffoldOptions{
				from:     "invalid image reference!",
				executor: "containerd-shim-spin",
			},
			expectedError: "invalid image reference provided: 'invalid image reference!'",
		},
		{
			name: "invalid autoscaler type",
			opts: ScaffoldOptions{
				from:       "ghcr.io/foo/example-app:v0.1.0",
				autoscaler: "invalid",
			},
			expectedError: "invalid autoscaler type 'invalid'; the autoscaler type must be either 'hpa' or 'keda'",
		},
		{
			name: "max replica count less than zero",
			opts: ScaffoldOptions{
				from:        "ghcr.io/foo/example-app:v0.1.0",
				autoscaler:  "hpa",
				maxReplicas: -1,
			},
			expectedError: "the maximum replica count (-1) must be equal to or greater than 0",
		},
		{
			name: "max replica count less than replica count",
			opts: ScaffoldOptions{
				from:        "ghcr.io/foo/example-app:v0.1.0",
				autoscaler:  "hpa",
				replicas:    5,
				maxReplicas: 2,
			},
			expectedError: "the minimum replica count (5) must be less than or equal to the maximum replica count (2)",
		},
		{
			name: "must set cpu limits for HPA",
			opts: ScaffoldOptions{
				from:       "ghcr.io/foo/example-app:v0.1.0",
				autoscaler: "hpa",
			},
			expectedError: "cpu limits must be set when autoscaling is enabled",
		},
		{
			name: "must set memory limits for HPA",
			opts: ScaffoldOptions{
				from:       "ghcr.io/foo/example-app:v0.1.0",
				autoscaler: "hpa",
				cpuLimit:   "50m",
			},
			expectedError: "memory limits must be set when autoscaling is enabled",
		},
		{
			name: "must set target cpu utilization percentage for HPA",
			opts: ScaffoldOptions{
				from:        "ghcr.io/foo/example-app:v0.1.0",
				autoscaler:  "hpa",
				cpuLimit:    "50m",
				memoryLimit: "100Mi",
			},
			expectedError: "target cpu utilization percentage (0) must be between 1 and 100",
		},
		{
			name: "must set target memory utilization percentage for HPA",
			opts: ScaffoldOptions{
				from:                           "ghcr.io/foo/example-app:v0.1.0",
				autoscaler:                     "hpa",
				cpuLimit:                       "50m",
				memoryLimit:                    "100Mi",
				targetCpuUtilizationPercentage: 1,
			},
			expectedError: "target memory utilization percentage (0) must be between 1 and 100",
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := scaffold(tc.opts)

			if tc.expectedError == "" {
				require.Nil(t, err)
			} else {
				require.NotNil(t, err)
				require.Equal(t, tc.expectedError, err.Error())
			}
		})
	}
}
