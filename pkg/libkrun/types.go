package libkrun

import (
	"gitlab.com/tozd/go/errors"

	"github.com/walteh/ec1/pkg/vmnet"
)

// Common errors
var (
	ErrLibkrunNotAvailable     = errors.New("libkrun not available")
	ErrVMNetNotAvailable       = errors.New("vmnet not available")
	ErrVMNetHelperNotAvailable = errors.New("vmnet helper not available")
	ErrNotImplemented          = errors.New("libkrun-go: not implemented")
)

// Context represents a libkrun configuration context (stub version)
type Context struct {
	id uint32
}

// LogLevel represents libkrun log levels
type LogLevel uint32

const (
	LogLevelOff   LogLevel = 0
	LogLevelError LogLevel = 1
	LogLevelWarn  LogLevel = 2
	LogLevelInfo  LogLevel = 3
	LogLevelDebug LogLevel = 4
	LogLevelTrace LogLevel = 5
)

// DiskFormat represents disk image formats
type DiskFormat uint32

const (
	DiskFormatRaw   DiskFormat = 0
	DiskFormatQcow2 DiskFormat = 1
)

// KernelFormat represents kernel formats
type KernelFormat uint32

const (
	KernelFormatRaw       KernelFormat = 0
	KernelFormatELF       KernelFormat = 1
	KernelFormatPEGZ      KernelFormat = 2
	KernelFormatImageBZ2  KernelFormat = 3
	KernelFormatImageGZ   KernelFormat = 4
	KernelFormatImageZSTD KernelFormat = 5
)

// VirglFlag represents virglrenderer flags
type VirglFlag uint32

const (
	VirglUseEGL          VirglFlag = 1 << 0
	VirglThreadSync      VirglFlag = 1 << 1
	VirglUseGLX          VirglFlag = 1 << 2
	VirglUseSurfaceless  VirglFlag = 1 << 3
	VirglUseGLES         VirglFlag = 1 << 4
	VirglUseExternalBlob VirglFlag = 1 << 5
	VirglVenus           VirglFlag = 1 << 6
	VirglNoVirgl         VirglFlag = 1 << 7
	VirglUseAsyncFenceCB VirglFlag = 1 << 8
	VirglRenderServer    VirglFlag = 1 << 9
	VirglDRM             VirglFlag = 1 << 10
)

// VMConfig represents basic VM configuration
type VMConfig struct {
	NumVCPUs uint8
	RAMMiB   uint32
}

// DiskConfig represents disk configuration
type DiskConfig struct {
	BlockID  string
	Path     string
	Format   DiskFormat
	ReadOnly bool
}

// VirtioFSConfig represents virtio-fs configuration
type VirtioFSConfig struct {
	Tag     string
	Path    string
	ShmSize *uint64 // nil for default
}

// NetworkConfig represents network configuration
type NetworkConfig struct {
	PasstFD     *int      // nil for TSI backend
	GvproxyPath *string   // nil for TSI backend
	MAC         *[6]uint8 // nil for auto-generated
	PortMap     []string  // host:guest port mappings
}

// GPUConfig represents GPU configuration
type GPUConfig struct {
	VirglFlags VirglFlag
	ShmSize    *uint64 // nil for default
}

// ProcessConfig represents process configuration
type ProcessConfig struct {
	ExecPath string
	Args     []string
	Env      []string
	WorkDir  *string // nil for default
}

// KernelConfig represents kernel configuration
type KernelConfig struct {
	Path      string
	Format    KernelFormat
	Initramfs *string // nil for no initramfs
	Cmdline   string
}

// VsockPort represents a vsock port mapping
type VsockPort struct {
	Port     uint32
	FilePath string
	Listen   *bool // nil for default behavior
}

// SecurityConfig represents security configuration
type SecurityConfig struct {
	UID              *uint32  // nil for current user
	GID              *uint32  // nil for current group
	Rlimits          []string // format: "RESOURCE=RLIM_CUR:RLIM_MAX"
	SMBIOSOEMStrings []string
}

// SEVConfig represents SEV-specific configuration
type SEVConfig struct {
	TEEConfigFile *string // Path to TEE configuration file
}

// AdvancedConfig represents advanced VM configuration
type AdvancedConfig struct {
	NestedVirt    *bool   // nil for default
	SoundDevice   *bool   // nil for default
	ConsoleOutput *string // nil for stdout
}

// VMNetConfig represents vmnet-specific network configuration
type VMNetConfig struct {
	// Basic vmnet options
	InterfaceID     *string             // nil for auto-generated
	OperationMode   vmnet.OperationMode // shared, bridged, or host
	StartAddress    *string             // nil for default (192.168.105.1)
	EndAddress      *string             // nil for default (192.168.105.254)
	SubnetMask      *string             // nil for default (255.255.255.0)
	SharedInterface *string             // required for bridged mode
	EnableIsolation *bool               // nil for default, used for host mode
	Verbose         bool                // enable verbose logging

	// Port mappings (same as NetworkConfig)
	PortMap []string // host:guest port mappings
}
