package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	spinv1 "github.com/spinkube/spin-operator/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/printers"
)

var (
	artifact string
	replicas int32
	dryRun   bool
)

var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "deploy spin app",
	RunE: func(cmd *cobra.Command, args []string) error {
		reference := strings.Split(artifact, ":")[0]
		referenceParts := strings.Split(reference, "/")
		name := referenceParts[len(referenceParts)-1]

		spinapp := spinv1.SpinApp{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
			TypeMeta: metav1.TypeMeta{
				APIVersion: "core.spinoperator.dev/v1",
				Kind:       "SpinApp",
			},
			Spec: spinv1.SpinAppSpec{
				Replicas: replicas,
				Image:    artifact,
				Executor: "containerd-shim-spin",
			},
		}

		if dryRun {
			y := printers.YAMLPrinter{}
			y.PrintObj(&spinapp, os.Stdout)
			return nil
		}

		err := k8simpl.ApplySpinApp(context.TODO(), &spinapp)
		if err != nil {
			return err
		}

		fmt.Printf("spinapp.spin.fermyon.com/%s configured\n", name)
		return nil
	},
}

func init() {
	deployCmd.Flags().BoolVar(&dryRun, "dry-run", false, "only print the SpinApp resource file without deploying")
	deployCmd.Flags().Int32VarP(&replicas, "replicas", "r", 2, "Number of replicas for the spin app")
	deployCmd.Flags().StringVarP(&artifact, "from", "f", "", "Reference in the registry of the Spin application")
	deployCmd.MarkFlagRequired("from")

	configFlags.AddFlags(deployCmd.Flags())
	rootCmd.AddCommand(deployCmd)
}
