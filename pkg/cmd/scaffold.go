package cmd

import (
	"log"
	"os"
	"strings"
	"text/template"

	"github.com/spf13/cobra"
)

type appConfig struct {
	Name     string
	Image    string
	Replicas int32
}

var manifestStr = `apiVersion: core.spinoperator.dev/v1
kind: SpinApp
metadata:
  name: {{ .Name }}
spec:
  image: "{{ .Image }}"
  replicas: {{ .Replicas }}
`

var scaffoldCmd = &cobra.Command{
	Use:   "scaffold",
	Short: "scaffold SpinApp manifest",
	RunE: func(cmd *cobra.Command, args []string) error {
		artifact, err := cmd.Flags().GetString("from")
		if err != nil {
			return err
		}

		reference := strings.Split(artifact, ":")[0]
		referenceParts := strings.Split(reference, "/")
		name := referenceParts[len(referenceParts)-1]

		replicas, err = cmd.Flags().GetInt32("replicas")
		if err != nil {
			return err
		}

		config := appConfig{
			Name:     name,
			Image:    artifact,
			Replicas: replicas,
		}

		tmpl, err := template.New("spinapp").Parse(manifestStr)
		if err != nil {
			panic(err)
		}

		output, err := cmd.Flags().GetString("out")
		if err != nil {
			return err
		}
		if output != "" {
			// Create a new file.
			file, err := os.Create(output)
			if err != nil {
				return err
			}
			defer file.Close()

			err = tmpl.Execute(file, config)
			if err != nil {
				return err
			}

			log.Printf("\nSpinApp manifest saved to %s\n", output)
			return nil

		}

		return tmpl.Execute(os.Stdout, config)
	},
}

func init() {
	scaffoldCmd.Flags().Int32P("replicas", "r", 2, "Number of replicas for the spin app")
	scaffoldCmd.Flags().StringP("from", "f", "", "Reference in the registry of the Spin application")
	scaffoldCmd.Flags().StringP("out", "o", "", "path to file to write manifest yaml")
	scaffoldCmd.MarkFlagRequired("from")

	rootCmd.AddCommand(scaffoldCmd)
}
