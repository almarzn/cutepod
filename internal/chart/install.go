package chart

import (
	"context"
	"cutepod/internal/container"
	"cutepod/internal/target"
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

	installTarget := target.NewInstallTarget(opts.Namespace)

	for name, chart := range charts {
		if v, ok := chart.(*container.CuteContainer); ok {
			fmt.Printf("Installing chart %s\n", name)
			err := v.Install(context.Background(), *installTarget)
			if err != nil {
				return err
			}
			continue
		}

		return fmt.Errorf("unknown chart type %s", name)
	}

	return nil
}
