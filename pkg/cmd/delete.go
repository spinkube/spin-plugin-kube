package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var deleteCmd = &cobra.Command{
	Use:    "delete app-name",
	Short:  "Delete app",
	Hidden: isExperimentalFlagNotSet,
	RunE: func(cmd *cobra.Command, args []string) error {
		var appName string
		if len(args) > 0 {
			appName = args[0]
		}

		if appName == "" {
			return fmt.Errorf("no app name specified to delete")
		}

		yes, err := cmd.Flags().GetBool("yes")
		if err != nil {
			return err
		}

		if !yes {
			yes, err = yesOrNo("This action is irreversible. Are you sure? (y/N): ")
			if err != nil {
				return err
			}
		}

		if !yes {
			return nil
		}

		okey := client.ObjectKey{
			Namespace: namespace,
			Name:      appName,
		}

		err = k8simpl.DeleteSpinApp(context.TODO(), okey)
		if err != nil {
			if apierrors.IsNotFound(err) {
				fmt.Printf("Could not find app with name %s\n", appName)
				os.Exit(1)
			}

			fmt.Println(err)
			os.Exit(1)
		}

		fmt.Printf("Succesfully deleted %s\n", appName)
		return nil
	},
}

func init() {
	configFlags.AddFlags(deleteCmd.Flags())

	deleteCmd.Flags().BoolP("yes", "y", false, "specify --yes to immediately delete the resource")
	rootCmd.AddCommand(deleteCmd)
}
