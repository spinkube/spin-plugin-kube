package cmd

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubectl/pkg/cmd/logs"
	"k8s.io/kubectl/pkg/cmd/portforward"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	"k8s.io/kubectl/pkg/polymorphichelpers"
)

const spinAppPort = "80"

var connectCmd = &cobra.Command{
	Use:   "connect [<app-name>]",
	Short: "connect to spin app locally",
	RunE: func(cmd *cobra.Command, args []string) error {
		var appName string
		if len(args) > 0 {
			appName = args[0]
		}

		if appName == "" && appNameFromCurrentDirContext != "" {
			appName = appNameFromCurrentDirContext
		}

		localPort, _ := cmd.Flags().GetString("local-port")
		fieldSelector, _ := cmd.Flags().GetString("field-selector")
		labelSelector, _ := cmd.Flags().GetString("label-selector")

		if appName == "" && fieldSelector == "" && labelSelector == "" {
			return fmt.Errorf("either one of app-name or fieldSelector or labelSelector is required")
		}

		getPodTimeout, err := cmdutil.GetPodRunningTimeoutFlag(cmd)
		if err != nil {
			return err
		}

		kubeclient, err := getKubernetesClientset()
		if err != nil {
			return err
		}

		resp, err := kubeclient.AppsV1().Deployments(namespace).List(context.TODO(), metav1.ListOptions{
			LabelSelector: labelSelector,
			FieldSelector: fieldSelector,
		})
		if err != nil {
			return err
		}

		if len(resp.Items) == 0 {
			return fmt.Errorf("no active deployment found for SpinApp")
		}

		var deploy appsv1.Deployment
		if appName != "" {
			for _, item := range resp.Items {
				if item.Name == appName {
					deploy = item
					break
				}
			}
		} else {
			deploy = resp.Items[0]
		}

		factory, streams := NewCommandFactory()
		pod, err := polymorphichelpers.AttachablePodForObjectFn(factory, &deploy, getPodTimeout)
		if err != nil {
			return err
		}

		fmt.Printf("connecting to pod %s/%s\n", pod.Namespace, pod.Name)
		reference := fmt.Sprintf("pod/%s", pod.Name)
		if strings.Contains(localPort, ":") {
			return fmt.Errorf("local port should not contain ':' character")
		}

		go func() {
			logOpts = logs.NewLogsOptions(streams, false)
			logOpts.Follow = true

			lccmd := logs.NewCmdLogs(factory, streams)

			cmdutil.CheckErr(logOpts.Complete(factory, lccmd, []string{reference}))
			cmdutil.CheckErr(logOpts.Validate())
			cmdutil.CheckErr(logOpts.RunLogs())
		}()

		ccmd := portforward.NewCmdPortForward(factory, streams)
		ccmd.Run(ccmd, []string{reference, fmt.Sprintf("%s:%s", localPort, spinAppPort)})

		return nil
	},
}

func init() {
	cmdutil.AddPodRunningTimeoutFlag(connectCmd, 30*time.Second)
	configFlags.AddFlags(connectCmd.Flags())

	connectCmd.Flags().StringP("local-port", "p", "", "local port to listen on when connecting to SpinApp")
	connectCmd.Flags().String("field-selector", "", "Selector (field query) to filter on, supports '=', '==', and '!='.(e.g. --field-selector key1=value1,key2=value2). The server only supports a limited number of field queries per type.")
	connectCmd.Flags().StringP("selector", "l", "", "Selector (label query) to filter on, supports '=', '==', and '!='.(e.g. -l key1=value1,key2=value2). Matching objects must satisfy all of the specified label constraints.")

	rootCmd.AddCommand(connectCmd)
}
