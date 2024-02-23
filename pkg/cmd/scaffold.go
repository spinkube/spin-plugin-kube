package cmd

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"
	"text/template"

	"github.com/spf13/cobra"
)

type ScaffoldOptions struct {
	from                              string
	replicas                          int32
	maxReplicas                       int32
	executor                          string
	output                            string
	configfile                        string
	enableAutoscaling                 bool
	cpuLimit                          string
	memoryLimit                       string
	cpuRequest                        string
	memoryRequest                     string
	targetCpuUtilizationPercentage    int32
	targetMemoryUtilizationPercentage int32
}

var scaffoldOpts = ScaffoldOptions{}

type appConfig struct {
	Name                              string
	Image                             string
	Executor                          string
	Replicas                          int32
	MaxReplicas                       int32
	RuntimeConfig                     string
	EnableAutoscaling                 bool
	CpuLimit                          string
	MemoryLimit                       string
	CpuRequest                        string
	MemoryRequest                     string
	TargetCpuUtilizationPercentage    int32
	TargetMemoryUtilizationPercentage int32
}

var manifestStr = `apiVersion: core.spinoperator.dev/v1
kind: SpinApp
metadata:
  name: {{ .Name }}
spec:
  image: "{{ .Image }}"
  executor: {{ .Executor }}
{{- if .EnableAutoscaling }}
  enableAutoscaling: true
{{- else }}
  replicas: {{ .Replicas }}
{{- end}}
{{- if or .CpuLimit .MemoryLimit }}
  resources:
    limits:
    {{- if .CpuLimit }}
      cpu: {{ .CpuLimit }}
    {{- end }}
    {{- if .MemoryLimit }}
      memory: {{ .MemoryLimit }}
    {{- end }}
{{- if or .CpuRequest .MemoryRequest }}
    requests:
    {{- if .CpuRequest }}
      cpu: {{ .CpuRequest }}
    {{- end }}
    {{- if .MemoryRequest }}
      memory: {{ .MemoryRequest }}
    {{- end }}
{{- end }}
{{- end }}
{{- if .RuntimeConfig }}
  runtimeConfig:
    loadFromSecret: {{ .Name }}-runtime-config
---
apiVersion: v1
kind: Secret
metadata:
  name: {{ .Name }}-runtime-config
type: Opaque
data:
  runtime-config.toml: {{ .RuntimeConfig }}
{{- end }}
{{- if .EnableAutoscaling }}
---
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: {{ .Name }}-autoscaler
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: {{ .Name }}
  minReplicas: {{ .Replicas }}
  maxReplicas: {{ .MaxReplicas }}
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: {{ .TargetCpuUtilizationPercentage }}
  - type: Resource
    resource:
     name: memory
      target:
        type: Utilization
        averageUtilization: {{ .TargetMemoryUtilizationPercentage }}
{{- end }}
`

var scaffoldCmd = &cobra.Command{
	Use:   "scaffold",
	Short: "scaffold SpinApp manifest",
	RunE: func(cmd *cobra.Command, args []string) error {
		content, err := scaffold(scaffoldOpts)
		if err != nil {
			return err
		}

		if scaffoldOpts.output != "" {
			err = os.WriteFile(scaffoldOpts.output, content, 0644)
			if err != nil {
				return err
			}

			log.Printf("\nSpinApp manifest saved to %s\n", scaffoldOpts.output)
			return nil

		}

		fmt.Fprint(os.Stdout, string(content))

		return nil
	},
}

func scaffold(opts ScaffoldOptions) ([]byte, error) {
	// flag validation

	// replica count must be greater than 0
	if opts.replicas < 0 {
		return nil, fmt.Errorf("replicas must be greater than 0")
	}

	// if autoscaling is enabled, max replicas must be set
	if opts.enableAutoscaling && opts.maxReplicas < 0 {
		return nil, fmt.Errorf("max replicas must be greater than 0")
	}

	// check that the image reference is valid
	if !validateImageReference(opts.from) {
		return nil, fmt.Errorf("invalid image reference")
	}

	// validate autoscaling flags
	//
	// NOTE: --replicas refers to the minimum number of replicas
	if opts.enableAutoscaling {
		// max replicas must be greater than 0
		if opts.maxReplicas < 0 {
			return nil, fmt.Errorf("max replicas must be greater than 0")
		}

		// max replicas must be greater than min replicas
		if opts.maxReplicas < opts.replicas {
			return nil, fmt.Errorf("max replicas must be equal to or greater than min replicas")
		}

		// cpu and memory limits must be set
		if opts.cpuLimit == "" || opts.memoryLimit == "" {
			return nil, fmt.Errorf("cpu and memory limits must be set when autoscaling is enabled")
		}

		// TODO: cpu and memory requests cannot exceed their respective limits

		// target cpu and memory utilization must be between 1 and 100
		if opts.targetCpuUtilizationPercentage < 1 || opts.targetCpuUtilizationPercentage > 100 {
			return nil, fmt.Errorf("target cpu utilization must be between 0 and 100")
		}

		if opts.targetMemoryUtilizationPercentage < 1 || opts.targetMemoryUtilizationPercentage > 100 {
			return nil, fmt.Errorf("target memory utilization must be between 0 and 100")
		}
	}

	reference := strings.Split(opts.from, ":")[0]
	referenceParts := strings.Split(reference, "/")
	name := referenceParts[len(referenceParts)-1]

	config := appConfig{
		Name:                              name,
		Image:                             opts.from,
		Replicas:                          opts.replicas,
		MaxReplicas:                       opts.maxReplicas,
		Executor:                          opts.executor,
		EnableAutoscaling:                 opts.enableAutoscaling,
		CpuLimit:                          opts.cpuLimit,
		MemoryLimit:                       opts.memoryLimit,
		CpuRequest:                        opts.cpuRequest,
		MemoryRequest:                     opts.memoryRequest,
		TargetCpuUtilizationPercentage:    opts.targetCpuUtilizationPercentage,
		TargetMemoryUtilizationPercentage: opts.targetMemoryUtilizationPercentage,
	}

	if opts.configfile != "" {
		raw, err := os.ReadFile(opts.configfile)
		if err != nil {
			return nil, err
		}

		config.RuntimeConfig = base64.StdEncoding.EncodeToString(raw)
	}

	tmpl, err := template.New("spinapp").Parse(manifestStr)
	if err != nil {
		return nil, err
	}

	var output bytes.Buffer
	err = tmpl.Execute(&output, config)
	if err != nil {
		return nil, err
	}

	return output.Bytes(), nil
}

func validateImageReference(imageRef string) bool {
	// This regex is designed to match strings that are valid image references, which include an optional registry (like
	// "ghcr.io"), a repository name (like "bacongobbler/hello-rust"), and an optional tag (like "1.0.0").
	//
	// The regex is quite complex, but in general it's looking for sequences of alphanumeric characters, separated by
	// periods, underscores, or hyphens, and optionally followed by a slash and more such sequences. The sequences can
	// be repeated any number of times. The final sequence can optionally be followed by a colon and another sequence,
	// representing the tag.
	pattern := `^([a-zA-Z0-9]+(?:[._-][a-zA-Z0-9]+)*/)*([a-zA-Z0-9]+(?:[._-][a-zA-Z0-9]+)*)?(:[a-zA-Z0-9]+(?:[._-][a-zA-Z0-9]+)*)?$|^([a-zA-Z0-9]+(?:[._-][a-zA-Z0-9]+)*/)*([a-zA-Z0-9]+(?:[._-][a-zA-Z0-9]+)*)?(:[a-zA-Z0-9]+(?:[._-][a-zA-Z0-9]+)*)?/([a-zA-Z0-9]+(?:[._-][a-zA-Z0-9]+)*)?(:[a-zA-Z0-9]+(?:[._-][a-zA-Z0-9]+)*)?$`
	regex := regexp.MustCompile(pattern)
	return regex.MatchString(imageRef)
}

func init() {
	scaffoldCmd.Flags().Int32VarP(&scaffoldOpts.replicas, "replicas", "r", 2, "Minimum number of replicas for the spin app")
	scaffoldCmd.Flags().Int32Var(&scaffoldOpts.maxReplicas, "max-replicas", 3, "Maximum number of replicas for the spin app. Autoscaling must be enabled to use this flag")
	scaffoldCmd.Flags().Int32Var(&scaffoldOpts.targetCpuUtilizationPercentage, "autoscaling-target-cpu-utilization", 60, "The target CPU utilization percentage to maintain across all pods")
	scaffoldCmd.Flags().Int32Var(&scaffoldOpts.targetMemoryUtilizationPercentage, "autoscaling-target-memory-utilization", 60, "The target memory utilization percentage to maintain across all pods")
	scaffoldCmd.Flags().BoolVar(&scaffoldOpts.enableAutoscaling, "enable-autoscaling", false, "Enable autoscaling support")
	scaffoldCmd.Flags().StringVar(&scaffoldOpts.executor, "executor", "containerd-shim-spin", "The executor used to run the Spin application")
	scaffoldCmd.Flags().StringVar(&scaffoldOpts.cpuLimit, "cpu-limit", "", "The maximum amount of CPU resource units the Spin application is allowed to use")
	scaffoldCmd.Flags().StringVar(&scaffoldOpts.cpuRequest, "cpu-request", "", "The amount of CPU resource units requested by the Spin application. Used to determine which node the Spin application will run on")
	scaffoldCmd.Flags().StringVar(&scaffoldOpts.memoryLimit, "memory-limit", "", "The maximum amount of memory the Spin application is allowed to use")
	scaffoldCmd.Flags().StringVar(&scaffoldOpts.memoryRequest, "memory-request", "", "The amount of memory requested by the Spin application. Used to determine which node the Spin application will run on")
	scaffoldCmd.Flags().StringVarP(&scaffoldOpts.from, "from", "f", "", "Reference in the registry of the Spin application")
	scaffoldCmd.Flags().StringVarP(&scaffoldOpts.output, "out", "o", "", "path to file to write manifest yaml")
	scaffoldCmd.Flags().StringVarP(&scaffoldOpts.configfile, "runtime-config-file", "c", "", "path to runtime config file")

	scaffoldCmd.MarkFlagRequired("from")

	rootCmd.AddCommand(scaffoldCmd)
}
