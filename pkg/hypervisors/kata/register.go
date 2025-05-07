package kata

import (
	"github.com/kata-containers/kata-containers/src/runtime/virtcontainers"
	"github.com/walteh/ec1/pkg/hypervisors"
)

func HypervisorRegistrationFunc(inter hypervisors.Hypervisor) func() virtcontainers.Hypervisor {
	return func() virtcontainers.Hypervisor {
		return &kataHypervisor{
			hypervisor: inter,
		}
	}
}
