package boot

import "context"

type BootConfigProvider interface {
	// isBootConfigProvider() bool
	Device(ctx context.Context) BootDevice
	Run(ctx context.Context) error
}

// fedora
type BootConfigurable interface {
	ConfigureBoot(ctx context.Context) (BootConfigProvider, error)
}
