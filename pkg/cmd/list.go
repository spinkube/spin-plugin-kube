package cmd

import (
	"context"
	"os"

	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:    "list",
	Short:  "List applications",
	Hidden: isExperimentalFlagNotSet,
	RunE: func(cmd *cobra.Command, args []string) error {
		appsResp, err := k8simpl.ListSpinApps(context.TODO(), namespace)
		if err != nil {
			return err
		}

		printApps(os.Stdout, appsResp.Items...)

		return nil
	},
}

func init() {
	configFlags.AddFlags(listCmd.Flags())
	rootCmd.AddCommand(listCmd)
}
