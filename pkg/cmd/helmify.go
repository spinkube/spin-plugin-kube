package cmd

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"

	_ "embed"

	"github.com/spf13/cobra"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chartutil"
)

// Default value for output flag
const defaultChartOutput string = "./charts"

//go:embed resources/.helmignore
var helmIgnore []byte

//go:embed resources/app.yaml
var appManifest []byte

//go:embed resources/hpa.yaml
var hpaManifest []byte

//go:embed resources/secret.yaml
var secretManifest []byte

//go:embed resources/scaledobject.yaml
var scaledObjectManifest []byte

//go:embed resources/NOTES.txt
var releaseNotes []byte

//go:embed resources/_helpers.tpl
var helpers []byte

var helmifyCmd = &cobra.Command{
	Use:   "helmify",
	Short: "Scaffold a Helm chart for your SpinApp",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if len(scaffoldOpts.output) == 0 {
			scaffoldOpts.output = defaultChartOutput
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		name, err := getNameFromImageReference(scaffoldOpts.from)
		if err != nil {
			return err
		}
		return scaffoldHelmChart(name, scaffoldOpts)
	},
}

func scaffoldHelmChart(name string, opts ScaffoldOptions) error {
	values, err := generateHelmValues(opts)
	if err != nil {
		return err
	}
	buf := new(bytes.Buffer)
	err = values.Encode(buf)
	if err != nil {
		return fmt.Errorf("could not encode values for Helm chart")
	}
	c := chart.Chart{
		Metadata: &chart.Metadata{
			Name:        name,
			Description: fmt.Sprintf("A Helm chart for %s", name),
			Type:        "application",
			APIVersion:  chart.APIVersionV2,
			AppVersion:  "0.1.0",
			Version:     "0.1.0",
		},
		Templates: []*chart.File{
			{
				Name: "values.yaml",
				Data: buf.Bytes(),
			},
			{
				Name: "templates/_helpers.tpl",
				Data: []byte(helpers),
			},
			{
				Name: "templates/app.yaml",
				Data: []byte(appManifest),
			},
			{
				Name: "templates/hpa.yaml",
				Data: []byte(hpaManifest),
			},
			{
				Name: "templates/scaledobject.yaml",
				Data: []byte(scaledObjectManifest),
			},
			{
				Name: "templates/secret.yaml",
				Data: []byte(secretManifest),
			},
			{
				Name: "templates/NOTES.txt",
				Data: []byte(releaseNotes),
			},
			{
				Name: ".helmignore",
				Data: []byte(helmIgnore),
			},
		},
		Values: values,
	}
	target := filepath.Base(opts.output)
	return chartutil.SaveDir(&c, target)
}

func generateHelmValues(opts ScaffoldOptions) (chartutil.Values, error) {
	values := chartutil.Values{
		"image":            opts.from,
		"replicas":         opts.replicas,
		"cpuLimit":         opts.cpuLimit,
		"memoryLimit":      opts.memoryLimit,
		"cpuRequest":       opts.cpuRequest,
		"memoryRequest":    opts.memoryRequest,
		"imagePullSecrets": opts.imagePullSecrets,
		"autoscaler":       opts.autoscaler,
	}
	if opts.configfile != "" {
		raw, err := os.ReadFile(opts.configfile)
		if err != nil {
			return nil, err
		}
		values["runtimeConfig"] = base64.StdEncoding.EncodeToString(raw)
	}
	return values, nil
}

func init() {
	// overwriting the default value of a flag defined on the parrent comand does not work
	// that's why I went for a PreRunE
	helmifyCmd.Flags().StringVarP(&scaffoldOpts.output, "out", "o", defaultChartOutput, "Path for writing the Helm chart to")
	scaffoldCmd.AddCommand(helmifyCmd)
}
