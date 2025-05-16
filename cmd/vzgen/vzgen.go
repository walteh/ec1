package main

import (
	"github.com/walteh/ec1/hypervisor"
)

//// go:generate go tool nswrap

func main() {
	// vzfobjc.Autoreleasepool(func() {
	// })

	_ = hypervisor.HvVmCreate
}
