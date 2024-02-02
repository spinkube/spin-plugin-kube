package cmd

import (
	"os"
	"strings"

	"github.com/pelletier/go-toml/v2"
	"github.com/spf13/cobra"
	"github.com/spinkube/spin-plugin-k8s/pkg/k8s"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

// global variables available to all sub-commands
var (
	appNameFromCurrentDirContext = ""
	configFlags                  = genericclioptions.NewConfigFlags(true)
	namespace                    string
	k8simpl                      *k8s.Impl
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = newRootCmd()

func newRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:     "k8s",
		Short:   "Manage apps running on Kubernetes",
		Version: Version,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			namespace = getNamespace(configFlags)
			k8sclient, err := getRuntimeClient()
			if err != nil {
				return err
			}

			k8simpl = k8s.New(k8sclient, configFlags)

			appNameFromCurrentDirContext, err = initAppNameFromCurrentDirContext()
			if err != nil {
				return err
			}

			return nil
		},
	}

	configFlags.AddFlags(rootCmd.Flags())
	return rootCmd
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

// getNamespace takes a set of kubectl flag values and returns the namespace we should be operating in
func getNamespace(flags *genericclioptions.ConfigFlags) string {
	namespace, _, err := flags.ToRawKubeConfigLoader().Namespace()
	if err != nil || len(namespace) == 0 {
		namespace = "default"
	}

	return namespace
}

func initAppNameFromCurrentDirContext() (string, error) {
	if strings.ToLower(os.Getenv("SPIN_K8S_DISABLE_DIR_CONTEXT")) == "true" {
		return "", nil
	}

	content, err := os.ReadFile("spin.toml")
	//running from a non spin-app dir
	if os.IsNotExist(err) {
		return "", nil
	}

	manifest := struct {
		Application struct {
			Name string `toml:"name"`
		} `toml:"application"`
	}{}

	err = toml.Unmarshal(content, &manifest)
	if err != nil {
		return "", err
	}

	return manifest.Application.Name, nil
}
