package port

import (
	"context"

	"gitlab.com/tozd/go/errors"
	"gvisor.dev/gvisor/pkg/rand"
	"gvisor.dev/gvisor/pkg/tcpip/ports"
)

var portManager *ports.PortManager

func lazyPortManager() *ports.PortManager {
	if portManager == nil {
		portManager = ports.NewPortManager()
	}
	return portManager
}

func ReservePort(ctx context.Context) (uint16, error) {

	reservation := ports.Reservation{
		Flags: (ports.FlagMask | ports.LoadBalancedFlag).ToFlags(),
		// Addr:  tcpip.AddrFromSlice(mac),
	}

	port, err := lazyPortManager().ReservePort(rand.RNGFrom(rand.Reader), reservation, nil)
	if err != nil {
		return 0, errors.Errorf("reserving port: %s", err.String()) // the error is not an error, so we can't use %w
	}

	go func() {
		<-ctx.Done()
		lazyPortManager().ReleasePort(reservation)
	}()

	return port, nil
}
