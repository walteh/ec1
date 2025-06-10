package ec1init

const (
	VsockPort             = 2019
	RealInitPath          = "/iniz"
	RootfsVirtioTag       = "rootfs"
	Ec1VirtioTag          = "ec1"
	Ec1AbsPath            = "/ec1"
	NewRootAbsPath        = "/newroot"
	VsockPidFile          = "/ec1.vsock.pid"
	ContainerCmdlineFile  = "/container-cmdline.json"
	ContainerManifestFile = "/container-manifest.json"
	ContainerSpecFile     = "/container-oci-spec.json"
	ContainerMountsFile   = "/container-mounts.json"
	ContainerTimesyncFile = "/timesync"
	ContainerReadyFile    = "/ready"
	TempVirtioTag         = "temp"
	DevConsoleLogPort     = "/dev/vport1p0"
	DevStdoutPort         = "/dev/vport2p0"
	DevStderrPort         = "/dev/vport3p0"
	DevStdinPort          = "/dev/vport4p0"
)
