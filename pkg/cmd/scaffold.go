package cmd

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"strings"
	"text/template"

	"github.com/spf13/cobra"
)

type ScaffoldOptions struct {
	from       string
	replicas   int32
	output     string
	configfile string
}

var scaffoldOpts = ScaffoldOptions{}

type appConfig struct {
	Name          string
	Image         string
	Replicas      int32
	RuntimeConfig string
}

var manifestStr = `apiVersion: core.spinoperator.dev/v1
kind: SpinApp
metadata:
  name: {{ .Name }}
spec:
  image: "{{ .Image }}"
  replicas: {{ .Replicas }}
  executor: containerd-shim-spin
{{- if .RuntimeConfig }}
  runtimeConfig:
    loadFromSecret: {{ .Name }}-runtime-config
  {{- end }}
{{ if .RuntimeConfig -}}
---
apiVersion: v1
kind: Secret
metadata:
  name: {{ .Name }}-runtime-config
type: Opaque
data:
  runtime-config.toml: {{ .RuntimeConfig }}
{{ end -}}
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
	reference := strings.Split(opts.from, ":")[0]
	referenceParts := strings.Split(reference, "/")
	name := referenceParts[len(referenceParts)-1]

	config := appConfig{
		Name:     name,
		Image:    opts.from,
		Replicas: opts.replicas,
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

func init() {
	scaffoldCmd.Flags().Int32VarP(&scaffoldOpts.replicas, "replicas", "r", 2, "Number of replicas for the spin app")
	scaffoldCmd.Flags().StringVarP(&scaffoldOpts.from, "from", "f", "", "Reference in the registry of the Spin application")
	scaffoldCmd.Flags().StringVarP(&scaffoldOpts.output, "out", "o", "", "path to file to write manifest yaml")
	scaffoldCmd.Flags().StringVarP(&scaffoldOpts.configfile, "runtime-config-file", "c", "", "path to runtime config file")

	scaffoldCmd.MarkFlagRequired("from")

	rootCmd.AddCommand(scaffoldCmd)
}
