package cmd

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"
	spinv1alpha1 "github.com/spinkube/spin-operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/printers"
)

var (
	artifact string
	replicas int32
	dryRun   bool
)

var deployCmd = &cobra.Command{
	Use:    "deploy",
	Short:  "Deploy application to Kubernetes",
	Hidden: isExperimentalFlagNotSet,
	RunE: func(_ *cobra.Command, _ []string) error {
		name, err := getNameFromImageReference(artifact)
		if err != nil {
			return err
		}

		spinapp := spinv1alpha1.SpinApp{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
			TypeMeta: metav1.TypeMeta{
				APIVersion: "core.spinkube.dev/v1alpha1",
				Kind:       "SpinApp",
			},
			Spec: spinv1alpha1.SpinAppSpec{
				Replicas: replicas,
				Image:    artifact,
				Executor: "containerd-shim-spin",
			},
		}

		if dryRun {
			y := printers.YAMLPrinter{}
			if err := y.PrintObj(&spinapp, os.Stdout); err != nil {
				return err
			}
			return nil
		}

		if err := kubeImpl.ApplySpinApp(context.TODO(), &spinapp); err != nil {
			return err
		}

		fmt.Printf("spinapp.spin.fermyon.com/%s configured\n", name)
		return nil
	},
}

func init() {
	deployCmd.Flags().BoolVar(&dryRun, "dry-run", false, "only print the kubernetes manifest without deploying")
	deployCmd.Flags().Int32VarP(&replicas, "replicas", "r", 2, "Number of replicas for the application")
	deployCmd.Flags().StringVarP(&artifact, "from", "f", "", "Reference in the registry of the application")

	if err := deployCmd.MarkFlagRequired("from"); err != nil {
		log.Fatal(err)
	}

	configFlags.AddFlags(deployCmd.Flags())
	rootCmd.AddCommand(deployCmd)
}
