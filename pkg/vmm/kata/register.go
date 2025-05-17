package kata

import (
	"github.com/kata-containers/kata-containers/src/runtime/virtcontainers"

	"github.com/walteh/ec1/pkg/vmm"
)

func HypervisorRegistrationFunc[VM vmm.VirtualMachine](inter vmm.Hypervisor[VM]) func() virtcontainers.Hypervisor {
	return func() virtcontainers.Hypervisor {
		return &kataHypervisor[VM]{
			hypervisor: inter,
		}
	}
}
