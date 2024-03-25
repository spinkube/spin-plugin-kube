package cmd

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"strings"
	"text/template"

	dockerparser "github.com/novln/docker-parser"
	"github.com/spf13/cobra"
)

type ScaffoldOptions struct {
	autoscaler                        string
	configfile                        string
	cpuLimit                          string
	cpuRequest                        string
	executor                          string
	from                              string
	imagePullSecrets                  []string
	maxReplicas                       int32
	memoryLimit                       string
	memoryRequest                     string
	output                            string
	replicas                          int32
	targetCpuUtilizationPercentage    int32
	targetMemoryUtilizationPercentage int32
}

var scaffoldOpts = ScaffoldOptions{}

type appConfig struct {
	Autoscaler                        string
	CpuLimit                          string
	CpuRequest                        string
	Executor                          string
	Image                             string
	ImagePullSecrets                  []string
	MaxReplicas                       int32
	MemoryLimit                       string
	MemoryRequest                     string
	Name                              string
	Replicas                          int32
	RuntimeConfig                     string
	TargetCpuUtilizationPercentage    int32
	TargetMemoryUtilizationPercentage int32
}

var manifestStr = `apiVersion: core.spinoperator.dev/v1alpha1
kind: SpinApp
metadata:
  name: {{ .Name }}
spec:
  image: "{{ .Image }}"
  executor: {{ .Executor }}
{{- if not (eq .Autoscaler "") }}
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
{{- if len .ImagePullSecrets }}
  imagePullSecrets:
{{- range $index, $secret := .ImagePullSecrets }}
    - name: {{ $secret -}}
{{ end }}
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
{{- if not (eq .Autoscaler "") }}
---
{{- if eq .Autoscaler "hpa" }}
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
{{- else if eq .Autoscaler "keda" }}
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: {{ .Name }}-autoscaler
spec:
  scaleTargetRef:
	apiVersion: apps/v1
	kind: Deployment
    name: {{ .Name }}
  minReplicaCount: {{ .Replicas }}
  maxReplicaCount: {{ .MaxReplicas }}
  triggers:
    - type: cpu
      metricType: Utilization
      metadata:
        value: "{{ .TargetCpuUtilizationPercentage }}"
	- type: memory
	  metricType: Utilization
	  metadata:
	    value: "{{ .TargetMemoryUtilizationPercentage }}"
{{- end }}
{{- end }}
`

var scaffoldCmd = &cobra.Command{
	Use:   "scaffold",
	Short: "Scaffold application manifest",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return validateScaffoldFlags(scaffoldOpts)
	},
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

			log.Printf("\nApplication manifest saved to %s\n", scaffoldOpts.output)
			return nil

		}

		fmt.Fprint(os.Stdout, string(content))
		return nil
	},
}

func validateScaffoldFlags(opts ScaffoldOptions) error {
	// flag validation

	// replica count must be greater than 0
	if opts.replicas < 0 {
		return fmt.Errorf("the minimum replica count (%d) must be greater than 0", opts.replicas)
	}

	// check that the image reference is valid
	if !validateImageReference(opts.from) {
		return fmt.Errorf("invalid image reference provided: '%s'", opts.from)
	}

	// validate autoscaling flags
	//
	// NOTE: --replicas refers to the minimum number of replicas
	if opts.autoscaler != "" {
		// autoscaler type must be a valid type
		if opts.autoscaler != "hpa" && opts.autoscaler != "keda" {
			return fmt.Errorf("invalid autoscaler type '%s'; the autoscaler type must be either 'hpa' or 'keda'", opts.autoscaler)
		}

		// max replicas must be equal to or greater than 0 (scale down to 0 replicas is allowed)
		if opts.maxReplicas < 0 {
			return fmt.Errorf("the maximum replica count (%d) must be equal to or greater than 0", opts.maxReplicas)
		}

		// min replicas must be less than or equal to max replicas
		if opts.replicas > opts.maxReplicas {
			return fmt.Errorf("the minimum replica count (%d) must be less than or equal to the maximum replica count (%d)", opts.replicas, opts.maxReplicas)
		}

		// cpu and memory limits must be set
		if opts.cpuLimit == "" {
			return fmt.Errorf("cpu limits must be set when autoscaling is enabled")
		}

		if opts.memoryLimit == "" {
			return fmt.Errorf("memory limits must be set when autoscaling is enabled")
		}

		// TODO: cpu and memory requests must be lower than their respective cpu/memory limit

		// target cpu and memory utilization must be between 1 and 100
		if opts.targetCpuUtilizationPercentage < 1 || opts.targetCpuUtilizationPercentage > 100 {
			return fmt.Errorf("target cpu utilization percentage (%d) must be between 1 and 100", opts.targetCpuUtilizationPercentage)
		}

		if opts.targetMemoryUtilizationPercentage < 1 || opts.targetMemoryUtilizationPercentage > 100 {
			return fmt.Errorf("target memory utilization percentage (%d) must be between 1 and 100", opts.targetMemoryUtilizationPercentage)
		}
	}

	return nil
}

func scaffold(opts ScaffoldOptions) ([]byte, error) {

	name, err := getNameFromImageReference(opts.from)
	if err != nil {
		return nil, err
	}

	config := appConfig{
		Name:                              name,
		Image:                             opts.from,
		Replicas:                          opts.replicas,
		MaxReplicas:                       opts.maxReplicas,
		Executor:                          opts.executor,
		CpuLimit:                          opts.cpuLimit,
		MemoryLimit:                       opts.memoryLimit,
		CpuRequest:                        opts.cpuRequest,
		MemoryRequest:                     opts.memoryRequest,
		TargetCpuUtilizationPercentage:    opts.targetCpuUtilizationPercentage,
		TargetMemoryUtilizationPercentage: opts.targetMemoryUtilizationPercentage,
		Autoscaler:                        opts.autoscaler,
		ImagePullSecrets:                  opts.imagePullSecrets,
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
	_, err := dockerparser.Parse(imageRef)
	return err == nil
}

func getNameFromImageReference(imageRef string) (string, error) {
	ref, err := dockerparser.Parse(imageRef)
	if err != nil {
		return "", err
	}

	if strings.Contains(ref.ShortName(), "/") {
		parts := strings.Split(ref.ShortName(), "/")
		return parts[len(parts)-1], nil
	}

	return ref.ShortName(), nil
}

func init() {
	scaffoldCmd.PersistentFlags().Int32VarP(&scaffoldOpts.replicas, "replicas", "r", 2, "Minimum number of replicas for the application")
	scaffoldCmd.PersistentFlags().Int32Var(&scaffoldOpts.maxReplicas, "max-replicas", 3, "Maximum number of replicas for the application. Autoscaling must be enabled to use this flag")
	scaffoldCmd.PersistentFlags().Int32Var(&scaffoldOpts.targetCpuUtilizationPercentage, "autoscaler-target-cpu-utilization", 60, "The target CPU utilization percentage to maintain across all pods")
	scaffoldCmd.PersistentFlags().Int32Var(&scaffoldOpts.targetMemoryUtilizationPercentage, "autoscaler-target-memory-utilization", 60, "The target memory utilization percentage to maintain across all pods")
	scaffoldCmd.PersistentFlags().StringVar(&scaffoldOpts.autoscaler, "autoscaler", "", "The autoscaler to use. Valid values are 'hpa' and 'keda'")
	scaffoldCmd.PersistentFlags().StringVar(&scaffoldOpts.executor, "executor", "containerd-shim-spin", "The executor used to run the application")
	scaffoldCmd.PersistentFlags().StringVar(&scaffoldOpts.cpuLimit, "cpu-limit", "", "The maximum amount of CPU resource units the application is allowed to use")
	scaffoldCmd.PersistentFlags().StringVar(&scaffoldOpts.cpuRequest, "cpu-request", "", "The amount of CPU resource units requested by the application. Used to determine which node the application will run on")
	scaffoldCmd.PersistentFlags().StringVar(&scaffoldOpts.memoryLimit, "memory-limit", "", "The maximum amount of memory the application is allowed to use")
	scaffoldCmd.PersistentFlags().StringVar(&scaffoldOpts.memoryRequest, "memory-request", "", "The amount of memory requested by the application. Used to determine which node the application will run on")
	scaffoldCmd.PersistentFlags().StringVarP(&scaffoldOpts.from, "from", "f", "", "Reference in the registry of the application")
	scaffoldCmd.PersistentFlags().StringVarP(&scaffoldOpts.configfile, "runtime-config-file", "c", "", "Path to runtime config file")
	scaffoldCmd.PersistentFlags().StringSliceVarP(&scaffoldOpts.imagePullSecrets, "image-pull-secret", "s", []string{}, "Secrets in the same namespace to use for pulling the image")
	scaffoldCmd.Flags().StringVarP(&scaffoldOpts.output, "out", "o", "", "Path to file to write manifest yaml")

	scaffoldCmd.MarkPersistentFlagRequired("from")

	rootCmd.AddCommand(scaffoldCmd)
}
