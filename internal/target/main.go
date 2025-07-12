package target

type InstallTarget struct {
	rootNamespace string
}

func NewInstallTarget(rootNamespace string) *InstallTarget {
	if rootNamespace == "" {
		rootNamespace = "default"
	}
	return &InstallTarget{rootNamespace}
}

func (i *InstallTarget) GetContainerName(objectNamespace string, objectName string) string {
	if objectNamespace == "" {
		objectNamespace = i.rootNamespace
	}
	return objectNamespace + "-" + objectName
}
