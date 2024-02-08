package cmd

import (
	"fmt"
	"io"

	"github.com/gosuri/uitable"
	spinv1 "github.com/spinkube/spin-operator/api/v1"
)

func printApps(w io.Writer, apps ...spinv1.SpinApp) {
	table := uitable.New()
	table.MaxColWidth = 50
	table.AddRow("NAMESPACE", "NAME", "EXECUTOR", "READY")

	for _, app := range apps {
		table.AddRow(app.Namespace, app.Name, app.Spec.Executor, fmt.Sprintf("%d/%d", app.Status.ReadyReplicas, app.Spec.Replicas))
	}

	fmt.Fprintln(w, table)
}
