package kube

import (
	"context"
	"fmt"

	spinv1alpha1 "github.com/spinkube/spin-operator/api/v1alpha1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const FieldManager = "spin-plugin-kube"

type Impl struct {
	kubeclient  client.Client
	configFlags *genericclioptions.ConfigFlags
}

func New(kubeclient client.Client, configFlags *genericclioptions.ConfigFlags) *Impl {
	return &Impl{
		kubeclient:  kubeclient,
		configFlags: configFlags,
	}
}

// ListSpinApps returns all resources of type SpinApp in the given namespace. If namespace is the empty string, it
// returns all SpinApp resources across all namespaces.
func (i *Impl) ListSpinApps(ctx context.Context, namespace string) (spinv1alpha1.SpinAppList, error) {
	var spinAppList spinv1alpha1.SpinAppList
	err := i.kubeclient.List(ctx, &spinAppList, &client.ListOptions{
		Namespace: namespace,
	})
	if err != nil {
		return spinv1alpha1.SpinAppList{}, err
	}

	return spinAppList, nil
}

func (i *Impl) ApplySpinApp(ctx context.Context, app *spinv1alpha1.SpinApp) error {
	patchMethod := client.Apply
	patchOptions := &client.PatchOptions{
		Force:        ptr(true),
		FieldManager: FieldManager,
	}

	return i.kubeclient.Patch(ctx, app, patchMethod, patchOptions)
}

func (i *Impl) GetSpinApp(ctx context.Context, name client.ObjectKey) (spinv1alpha1.SpinApp, error) {
	var app spinv1alpha1.SpinApp
	err := i.kubeclient.Get(ctx, name, &app)
	if err != nil {
		return spinv1alpha1.SpinApp{}, err
	}

	return app, nil
}

func (i *Impl) DeleteSpinApp(ctx context.Context, name client.ObjectKey) error {
	app, err := i.GetSpinApp(ctx, name)
	if err != nil {
		return err
	}

	fmt.Println("calling delete")
	err = i.kubeclient.Delete(ctx, &app)
	if err != nil {
		return err
	}

	return nil
}

func ptr[T any](v T) *T {
	return &v
}
