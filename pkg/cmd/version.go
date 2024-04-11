package cmd

import (
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"
	"k8s.io/client-go/discovery"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

// Version is set during build time
var Version = "unknown"

// actionCmd is the github action command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Display version information",
	Run: func(cmd *cobra.Command, _ []string) {
		shortFlag, err := cmd.Flags().GetBool("short")
		if err != nil {
			log.Fatalf("Error getting short flag: %v", err)
		}

		if shortFlag {
			fmt.Println(Version)
			return
		}

		spinVersion := os.Getenv("SPIN_VERSION")
		printVersionLine("Plugin Version", Version)
		if spinVersion != "" {
			printVersionLine("Spin Version", "v"+spinVersion)
		}

		serverVersion, err := getServerVersion()
		if err != nil {
			return
		}
		printVersionLine("Kubernetes Version", serverVersion)
	},
}

func printVersionLine(name string, version string) {
	fmt.Printf("%-14s: %s\n", name, version)
}

func getServerVersion() (string, error) {
	cfg, err := config.GetConfig()
	if err != nil {
		return "", err
	}

	client, err := discovery.NewDiscoveryClientForConfig(cfg)
	if err != nil {
		return "", err
	}

	serverVersion, err := client.ServerVersion()
	if err != nil {
		return "", err
	}

	return serverVersion.String(), nil
}

func init() {
	versionCmd.Flags().BoolP("short", "s", false, "Print only the plugin version")
	rootCmd.AddCommand(versionCmd)
}
