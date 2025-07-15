package object

type InstallTarget struct {
	rootNamespace string
}

func NewInstallTarget(rootNamespace string) *InstallTarget {
	if rootNamespace == "" {
		rootNamespace = "default"
	}
	return &InstallTarget{rootNamespace}
}

func (i *InstallTarget) GetContainerName(d Describe) string {
	namespace := i.GetNamespace()
	return namespace + "-" + d.GetName()
}

func (i *InstallTarget) GetNamespace() string {
	return i.rootNamespace
}
