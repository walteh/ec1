package main

/*
#include <spawn.h>
#include <stdlib.h>
extern char **environ;

// Helper to call posix_spawn.
static int spawn_helper(pid_t *pid,
                        const char *path,
                        char *const argv[],
                        char *const envp[]) {
    return posix_spawn(pid, path, NULL, NULL, argv, envp);
}
*/
import "C"
import (
	"fmt"
	"unsafe"
)

// Spawn executes path with args, returning the child PID.
func Spawn(path string, args []string) (int, error) {
	cPath := C.CString(path)
	defer C.free(unsafe.Pointer(cPath))

	// Build argv array
	cArgv := make([]*C.char, len(args)+2)
	cArgv[0] = cPath
	for i, a := range args {
		cs := C.CString(a)
		defer C.free(unsafe.Pointer(cs))
		cArgv[i+1] = cs
	}
	cArgv[len(args)+1] = nil

	// Call the C helper
	var pid C.pid_t
	ret := C.spawn_helper(&pid,
		cPath,
		(**C.char)(unsafe.Pointer(&cArgv[0])),
		C.environ)
	if ret != 0 {
		return 0, fmt.Errorf("posix_spawn failed: %d", ret)
	}
	return int(pid), nil
}
