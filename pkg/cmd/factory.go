package cmd

import (
	"os"

	spinv1 "github.com/spinkube/spin-operator/api/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func NewCommandFactory() (cmdutil.Factory, genericclioptions.IOStreams) {
	matchVersionKubeConfigFlags := cmdutil.NewMatchVersionFlags(configFlags)
	return cmdutil.NewFactory(matchVersionKubeConfigFlags),
		genericclioptions.IOStreams{In: os.Stdin, Out: os.Stdout, ErrOut: os.Stderr}
}

func getRuntimeClient() (client.Client, error) {
	var scheme = runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(spinv1.AddToScheme(scheme))

	config, err := configFlags.ToRESTConfig()
	if err != nil {
		return nil, err
	}

	return client.New(config, client.Options{
		Scheme: scheme,
	})
}

func getKubernetesClientset() (kubernetes.Interface, error) {
	config, err := configFlags.ToRESTConfig()
	if err != nil {
		return nil, err
	}

	return kubernetes.NewForConfig(config)
}
