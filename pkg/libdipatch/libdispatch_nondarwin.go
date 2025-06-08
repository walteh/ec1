//go:build !darwin

package libdispatch

// DispatchMain is a no-op on non-Darwin platforms,
// since libdispatch may not be installed or behaves differently.
func DispatchMain() {
	panic("DispatchMain is not supported on non-Darwin platforms")
}
