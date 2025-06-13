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

package options

// Options for the runm VM runtime
type Options struct {
	// Root is the root directory for VM state
	Root string `json:"root,omitempty"`

	// VMType specifies the type of VM to use (e.g., "vz" for Apple Virtualization)
	VMType string `json:"vm_type,omitempty"`

	// Memory is the amount of memory to allocate to the VM in MB
	Memory uint64 `json:"memory,omitempty"`

	// CPUs is the number of CPUs to allocate to the VM
	CPUs uint32 `json:"cpus,omitempty"`

	// IoUID is the UID for I/O operations
	IoUID uint32 `json:"io_uid,omitempty"`

	// IoGID is the GID for I/O operations
	IoGID uint32 `json:"io_gid,omitempty"`

	// Debug enables debug mode
	Debug bool `json:"debug,omitempty"`

	// ShimCgroup is the cgroup for the shim process
	ShimCgroup string `json:"shim_cgroup,omitempty"`
}

// CheckpointOptions for VM checkpoint operations
type CheckpointOptions struct {
	// Exit causes the VM to exit after checkpoint
	Exit bool `json:"exit,omitempty"`

	// Path is the path to save the checkpoint
	Path string `json:"path,omitempty"`

	// WorkPath is the working directory for checkpoint operations
	WorkPath string `json:"work_path,omitempty"`
}
