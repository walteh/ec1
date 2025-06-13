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

package runm

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/walteh/ec1/pkg/vmm/options"
)

const optionsFilename = "options.json"

// VM represents a virtual machine runtime
type VM struct {
	path      string
	namespace string
	options   *options.Options
}

// NewVM creates a new VM instance
func NewVM(path, namespace string, opts *options.Options) *VM {
	return &VM{
		path:      path,
		namespace: namespace,
		options:   opts,
	}
}

// Delete removes the VM and cleans up resources
func (vm *VM) Delete(ctx context.Context, id string) error {
	// For now, this is a placeholder
	// In the future, this would integrate with our actual VM management
	// to stop and clean up the VM instance
	return nil
}

// ReadOptions reads the option information from the path.
// When the file does not exist, ReadOptions returns nil without an error.
func ReadOptions(path string) (*options.Options, error) {
	filePath := filepath.Join(path, optionsFilename)
	if _, err := os.Stat(filePath); err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	var opts options.Options
	if err := json.Unmarshal(data, &opts); err != nil {
		return nil, err
	}
	return &opts, nil
}

// WriteOptions writes the options information into the path
func WriteOptions(path string, opts *options.Options) error {
	data, err := json.Marshal(opts)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(path, optionsFilename), data, 0600)
}
