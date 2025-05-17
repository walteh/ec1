package applevf

type Options struct {
	Vcpus     uint
	MemoryMiB uint

	VmlinuzPath   string
	KernelCmdline string
	InitrdPath    string

	// Bootloader stringSliceValue

	TimeSync string

	Devices []string

	RestfulURI string

	LogLevel string

	UseGUI bool

	IgnitionPath string

	// CloudInitFiles stringSliceValue
}
