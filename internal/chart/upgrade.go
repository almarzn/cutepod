package chart

import (
	"context"
	"cutepod/internal/container"
	"cutepod/internal/object"
	"fmt"
	"maps"
	"sort"
	"time"

	"github.com/briandowns/spinner"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/tree"
)

type UpgradeOptions struct {
	ChartPath string
	Namespace string
	DryRun    bool
	Verbose   bool
}

var (
	kindStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#f8fafc"))
	nameStyle    = lipgloss.NewStyle().Bold(true)
	addStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#22c55e")) // Green
	noneStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#94a3b8")) // Green
	updateStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#60a5fa")) // Yellow
	removeStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#ef4444")) // Red
	doneStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("70"))
	failStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	detailsStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#86198f"))
)

func Upgrade(opts UpgradeOptions) error {
	charts, err := Parse(ParseOptions{
		ChartPath: opts.ChartPath,
		Namespace: opts.Namespace,
		Verbose:   opts.Verbose,
	})
	if err != nil {
		return fmt.Errorf("parse error: %w", err)
	}

	installTarget := object.NewInstallTarget(opts.Namespace)
	ctx := context.Background()

	changes, err := getContainerChanges(ctx, installTarget, charts)
	if err != nil {
		return fmt.Errorf("change detection error: %w", err)
	}

	if len(changes) == 0 {
		fmt.Println("✅ No changes to apply.")
		return nil
	}
	printChangeTreeTreeStyle(changes)

	if opts.DryRun {
		fmt.Println(lipgloss.NewStyle().Bold(true).Render("🧪 Dry Run: No changes was applied.\n"))
		return nil
	}

	fmt.Println(lipgloss.NewStyle().Bold(true).Render("🚀 Applying changes...\n"))

	successCount := 0
	failCount := 0

	for e := range maps.Values(changes) {
		for _, c := range e {
			// Create and start spinner with object name
			s := spinner.New(spinner.CharSets[9], 100*time.Millisecond)
			s.Suffix = " " + c.GetName() + "..."
			s.Start()

			// Execute the object
			err := c.Execute(ctx)

			// Stop spinner before printing result
			s.Stop()

			if err != nil {
				failCount++
				fmt.Println(failStyle.Bold(true).Render("✕ ") + c.GetName() + ": " + err.Error())
			} else {
				successCount++
				fmt.Println(doneStyle.Bold(true).Render("✓ ") + c.GetName())
			}
		}
	}

	fmt.Printf("\n%s\n", lipgloss.NewStyle().Bold(true).Render(fmt.Sprintf("🎉 Upgrade complete: %d succeeded, %d failed", successCount, failCount)))
	return nil
}

func getObjectType(c string) string {
	switch c {
	case "CuteContainer":
		return "📦 containers: "
	}
	return ""
}
func printChangeTreeTreeStyle(grouped map[string][]object.Change) {
	// same styles...
	enumeratorStyle := noneStyle
	itemStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("212")).MarginLeft(1)
	emojiMap := map[string]string{
		"Add":    addStyle.Render("[+]"),
		"Update": updateStyle.Render("[~]"),
		"Remove": removeStyle.Render("[-]"),
		"None":   noneStyle.Render("[✓]"),
	}
	colorMap := map[string]lipgloss.Style{
		"Add":    addStyle,
		"Update": updateStyle,
		"Remove": removeStyle,
		"None":   noneStyle,
	}

	for kind, list := range grouped {
		t := tree.
			Root(getObjectType(kind)).
			Enumerator(tree.DefaultEnumerator).
			EnumeratorStyle(enumeratorStyle).
			RootStyle(kindStyle).
			ItemStyle(itemStyle)

		sort.Slice(list, func(i, j int) bool {
			return list[i].GetName() < list[j].GetName()
		})

		maxNameLen := 0
		for _, c := range list {
			if l := len(c.GetName()); l > maxNameLen {
				maxNameLen = l
			}
		}

		for _, c := range list {
			emoji := emojiMap[c.GetKind()]
			paddedName := fmt.Sprintf("%-*s", maxNameLen, c.GetName())
			line := fmt.Sprintf("%s %s  %s", emoji, colorMap[c.GetKind()].Render(paddedName), detailsStyle.Render(c.Describe()))
			t = t.Child(line)
		}

		fmt.Println(t.String() + "\n")

	}
}

func getContainerChanges(ctx context.Context, installTarget *object.InstallTarget, charts map[string]object.Actions) (map[string][]object.Change, error) {
	cs := make([]container.CuteContainer, 0)

	for _, c := range charts {
		if e, ok := c.(*container.CuteContainer); ok {
			cs = append(cs, *e)
		}
	}

	changes, err := container.GetChanges(ctx, *installTarget, cs)
	if err != nil {
		return nil, err
	}
	if len(changes) == 0 {
		return map[string][]object.Change{}, nil
	}

	return map[string][]object.Change{
		"CuteContainer": changes,
	}, nil
}
