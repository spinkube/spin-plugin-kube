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

	dockerparser "github.com/novln/docker-parser"
	"github.com/spf13/cobra"
)

type ScaffoldOptions struct {
	name                              string
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
	targetCPUUtilizationPercentage    int32
	targetMemoryUtilizationPercentage int32
	variables                         map[string]string
	components                        []string
}

var scaffoldOpts = ScaffoldOptions{}

type appConfig struct {
	Autoscaler                        string
	CPULimit                          string
	CPURequest                        string
	Executor                          string
	Image                             string
	ImagePullSecrets                  []string
	MaxReplicas                       int32
	MemoryLimit                       string
	MemoryRequest                     string
	Name                              string
	Replicas                          int32
	RuntimeConfig                     string
	TargetCPUUtilizationPercentage    int32
	TargetMemoryUtilizationPercentage int32
	Variables                         map[string]string
	Components                        []string
}

var manifestStr = `apiVersion: core.spinkube.dev/v1alpha1
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
{{- if .Variables }}
  variables:
{{- range $key, $value := .Variables }}
  - name: {{ $key }}
    value: {{ $value }}
{{- end }}
{{- end }}
{{- if .Components }}
  components:
{{- range $c := .Components }}
  - {{ $c }}
{{- end }}
{{- end }}
{{- if or .CPULimit .MemoryLimit }}
  resources:
    limits:
    {{- if .CPULimit }}
      cpu: {{ .CPULimit }}
    {{- end }}
    {{- if .MemoryLimit }}
      memory: {{ .MemoryLimit }}
    {{- end }}
{{- if or .CPURequest .MemoryRequest }}
    requests:
    {{- if .CPURequest }}
      cpu: {{ .CPURequest }}
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
        averageUtilization: {{ .TargetCPUUtilizationPercentage }}
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
      value: "{{ .TargetCPUUtilizationPercentage }}"
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
	RunE: func(_ *cobra.Command, _ []string) error {
		content, err := scaffold(scaffoldOpts)
		if err != nil {
			return err
		}

		if scaffoldOpts.output != "" {
			err = os.WriteFile(scaffoldOpts.output, content, 0600)
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

func validateFlags(opts ScaffoldOptions) error {
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
		if opts.targetCPUUtilizationPercentage < 1 || opts.targetCPUUtilizationPercentage > 100 {
			return fmt.Errorf("target cpu utilization percentage (%d) must be between 1 and 100", opts.targetCPUUtilizationPercentage)
		}

		if opts.targetMemoryUtilizationPercentage < 1 || opts.targetMemoryUtilizationPercentage > 100 {
			return fmt.Errorf("target memory utilization percentage (%d) must be between 1 and 100", opts.targetMemoryUtilizationPercentage)
		}
	}
	return nil
}

func scaffold(opts ScaffoldOptions) ([]byte, error) {
	if err := validateFlags(opts); err != nil {
		return nil, err
	}
	name, err := getNameFromImageReference(opts.from)
	if err != nil {
		return nil, err
	}
	if len(opts.name) > 0 {
		if !validateName(opts.name) {
			return nil, fmt.Errorf("invalid name provided. Must be a valid DNS subdomain name and not more than 253 chars")
		}
		name = opts.name
	}

	config := appConfig{
		Name:                              name,
		Image:                             opts.from,
		Replicas:                          opts.replicas,
		MaxReplicas:                       opts.maxReplicas,
		Executor:                          opts.executor,
		CPULimit:                          opts.cpuLimit,
		MemoryLimit:                       opts.memoryLimit,
		CPURequest:                        opts.cpuRequest,
		MemoryRequest:                     opts.memoryRequest,
		TargetCPUUtilizationPercentage:    opts.targetCPUUtilizationPercentage,
		TargetMemoryUtilizationPercentage: opts.targetMemoryUtilizationPercentage,
		Autoscaler:                        opts.autoscaler,
		ImagePullSecrets:                  opts.imagePullSecrets,
		Variables:                         opts.variables,
		Components:                        opts.components,
	}

	if opts.configfile != "" {
		raw, readErr := os.ReadFile(opts.configfile)
		if readErr != nil {
			return nil, readErr
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

func validateName(name string) bool {
	// ensure name is a valid DNS subdomain
	const pattern = `^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$`

	// check length
	if len(name) < 1 || len(name) > 253 {
		return false
	}

	// Compile the regex
	re := regexp.MustCompile(pattern)

	// Match the name against the pattern
	return re.MatchString(name)
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
	scaffoldCmd.Flags().Int32VarP(&scaffoldOpts.replicas, "replicas", "r", 2, "Minimum number of replicas for the application")
	scaffoldCmd.Flags().Int32Var(&scaffoldOpts.maxReplicas, "max-replicas", 3, "Maximum number of replicas for the application. Autoscaling must be enabled to use this flag")
	scaffoldCmd.Flags().Int32Var(&scaffoldOpts.targetCPUUtilizationPercentage, "autoscaler-target-cpu-utilization", 60, "The target CPU utilization percentage to maintain across all pods")
	scaffoldCmd.Flags().Int32Var(&scaffoldOpts.targetMemoryUtilizationPercentage, "autoscaler-target-memory-utilization", 60, "The target memory utilization percentage to maintain across all pods")
	scaffoldCmd.Flags().StringVar(&scaffoldOpts.autoscaler, "autoscaler", "", "The autoscaler to use. Valid values are 'hpa' and 'keda'")
	scaffoldCmd.Flags().StringVar(&scaffoldOpts.executor, "executor", "containerd-shim-spin", "The executor used to run the application")
	scaffoldCmd.Flags().StringVar(&scaffoldOpts.cpuLimit, "cpu-limit", "", "The maximum amount of CPU resource units the application is allowed to use")
	scaffoldCmd.Flags().StringVar(&scaffoldOpts.cpuRequest, "cpu-request", "", "The amount of CPU resource units requested by the application. Used to determine which node the application will run on")
	scaffoldCmd.Flags().StringVar(&scaffoldOpts.memoryLimit, "memory-limit", "", "The maximum amount of memory the application is allowed to use")
	scaffoldCmd.Flags().StringVar(&scaffoldOpts.memoryRequest, "memory-request", "", "The amount of memory requested by the application. Used to determine which node the application will run on")
	scaffoldCmd.Flags().StringVarP(&scaffoldOpts.from, "from", "f", "", "Reference in the registry of the application")
	scaffoldCmd.Flags().StringVarP(&scaffoldOpts.output, "out", "o", "", "Path to file to write manifest yaml")
	scaffoldCmd.Flags().StringVarP(&scaffoldOpts.configfile, "runtime-config-file", "c", "", "Path to runtime config file")
	scaffoldCmd.Flags().StringSliceVarP(&scaffoldOpts.imagePullSecrets, "image-pull-secret", "s", []string{}, "Secrets in the same namespace to use for pulling the image")
	scaffoldCmd.PersistentFlags().StringToStringVarP(&scaffoldOpts.variables, "variable", "v", nil, "Application variable (name=value) to be provided to the application")
	scaffoldCmd.PersistentFlags().StringSliceVarP(&scaffoldOpts.components, "component", "", nil, "Component ID to run. This can be specified multiple times. The default is all components.")
	scaffoldCmd.Flags().StringVarP(&scaffoldOpts.name, "name", "", "", "Overwrite the generated name of the application")
	if err := scaffoldCmd.MarkFlagRequired("from"); err != nil {
		log.Fatal(err)
	}

	rootCmd.AddCommand(scaffoldCmd)
}
