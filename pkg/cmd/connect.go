package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rajatjindal/kubectl-reverse-proxy/pkg/proxy"
	"github.com/spf13/cobra"
)

var connectCmd = &cobra.Command{
	Use:    "connect [<app-name>]",
	Short:  "connect to spin app locally",
	Hidden: isExperimentalFlagNotSet,
	RunE: func(cmd *cobra.Command, args []string) error {
		var appName string
		if len(args) > 0 {
			appName = args[0]
		}

		if appName == "" && appNameFromCurrentDirContext != "" {
			appName = appNameFromCurrentDirContext
		}

		if appName == "" {
			return fmt.Errorf("app name is required")
		}

		port, err := cmd.Flags().GetString("local-port")
		if err != nil {
			return err
		}

		adminPort, err := cmd.Flags().GetString("admin-port")
		if err != nil {
			return err
		}

		k8sclient, err := getKubernetesClientset()
		if err != nil {
			return err
		}

		stopCh := make(chan struct{})
		factory, streams := NewCommandFactory()

		config := &proxy.Config{
			K8sClient:     k8sclient,
			LabelSelector: fmt.Sprintf("kubernetes.io/service-name=%s", appName),
			Namespace:     getNamespace(configFlags),
			ListenPort:    port,
			AdminPort:     adminPort,
			Factory:       factory,
			Streams:       streams,
			StopCh:        stopCh,
		}

		fmt.Printf("starting reverse proxy listening on localhost:%s\n", port)

		// starts in background
		proxy.Start(cmd.Context(), config)

		sigterm := make(chan os.Signal, 1)
		signal.Notify(sigterm, syscall.SIGTERM)
		signal.Notify(sigterm, syscall.SIGINT)
		<-sigterm

		close(stopCh)
		fmt.Println("Stopping proxy. Press Ctrl+C again to kill immediately")

		for {
			select {
			case <-sigterm:
				return nil
			case <-time.NewTicker(2 * time.Second).C:
				return nil
			}
		}
	},
}

func init() {
	connectCmd.Flags().StringP("local-port", "p", "8081", "local port to start proxy on")
	connectCmd.Flags().StringP("admin-port", "a", "2019", "reverse proy admin port")

	configFlags.AddFlags(connectCmd.Flags())
	rootCmd.AddCommand(connectCmd)
}
