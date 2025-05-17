package kata

import (
	"github.com/kata-containers/kata-containers/src/runtime/virtcontainers"

	"github.com/walteh/ec1/pkg/hypervisors"
)

func HypervisorRegistrationFunc[VM hypervisors.VirtualMachine](inter hypervisors.Hypervisor[VM]) func() virtcontainers.Hypervisor {
	return func() virtcontainers.Hypervisor {
		return &kataHypervisor[VM]{
			hypervisor: inter,
		}
	}
}
