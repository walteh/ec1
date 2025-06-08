//go:build darwin

package libdispatch

/*
#cgo darwin LDFLAGS: -framework Foundation
#include <dispatch/dispatch.h>
*/
import "C"

// DispatchMain parks the calling thread and services the main GCD queue.
// On macOS this invokes dispatch_main().
func DispatchMain() {
	C.dispatch_main()
}
