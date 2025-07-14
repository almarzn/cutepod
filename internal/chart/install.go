package chart

import (
	"context"
	"cutepod/internal/object"
	"fmt"
)

type InstallOptions struct {
	ChartPath string
	Namespace string
	DryRun    bool
	Verbose   bool
}

// Install  chart templates
func Install(opts InstallOptions) error {
	charts, err := Parse(ParseOptions{
		ChartPath: opts.ChartPath,
		Namespace: opts.Namespace,
		Verbose:   opts.Verbose,
	})

	if err != nil {
		fmt.Println(err)
		return err
	}

	installTarget := object.NewInstallTarget(opts.Namespace)

	for name, chart := range charts {
		fmt.Printf("Installing chart %s\n", name)
		err := chart.Install(context.Background(), *installTarget)
		if err != nil {
			return err
		}
	}

	return nil
}
