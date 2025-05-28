package ec1init

const (
	VsockPort             = 2019
	RealInitPath          = "/iniz"
	RootfsVirtioTag       = "rootfs"
	Ec1VirtioTag          = "ec1"
	Ec1AbsPath            = "/ec1"
	NewRootAbsPath        = "/newroot"
	VsockPidFile          = "/ec1.vsock.pid"
	UserProvidedCmdline   = "/ec1.container-cmd.json"
	ContainerManifestFile = "/ec1.manifest.json"
)
