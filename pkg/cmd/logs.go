package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"k8s.io/kubectl/pkg/cmd/logs"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
)

var logOpts *logs.LogsOptions

var logsCmd = &cobra.Command{
	Use:    "logs [<app-name>]",
	Short:  "print the logs for a SpinApp",
	Hidden: os.Getenv("SPIN_EXPERIMENTAL") == "",
	Run: func(cmd *cobra.Command, args []string) {
		var appName string
		if len(args) > 0 {
			appName = args[0]
		}

		if appName == "" && appNameFromCurrentDirContext != "" {
			appName = appNameFromCurrentDirContext
		}

		reference := fmt.Sprintf("deployment/%s", appName)

		factory, streams := NewCommandFactory()
		ccmd := logs.NewCmdLogs(factory, streams)

		cmdutil.CheckErr(logOpts.Complete(factory, ccmd, []string{reference}))
		cmdutil.CheckErr(logOpts.Validate())
		cmdutil.CheckErr(logOpts.RunLogs())
	},
}

func init() {
	_, streams := NewCommandFactory()
	logOpts = logs.NewLogsOptions(streams, false)
	logOpts.AddFlags(logsCmd)

	configFlags.AddFlags(logsCmd.Flags())
	rootCmd.AddCommand(logsCmd)
}
