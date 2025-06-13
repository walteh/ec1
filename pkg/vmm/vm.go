/*
   Copyright The containerd Authors.

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package vmm

import (
	"context"
)

// VM represents a virtual machine instance
type VM interface {
	// Start starts the VM
	Start(ctx context.Context) error

	// Stop stops the VM
	Stop(ctx context.Context) error

	// Delete removes the VM and cleans up resources
	Delete(ctx context.Context) error

	// Kill sends a signal to the VM
	Kill(ctx context.Context, signal uint32) error

	// Wait waits for the VM to exit
	Wait(ctx context.Context) error

	// Status returns the current status of the VM
	Status(ctx context.Context) (string, error)

	// Pid returns the process ID of the VM (if applicable)
	Pid() int
}
