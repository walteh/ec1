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

package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"time"

	taskAPI "github.com/containerd/containerd/api/runtime/task/v3"
	"github.com/containerd/containerd/api/types"
	"github.com/containerd/containerd/v2/core/mount"
	"github.com/containerd/containerd/v2/pkg/namespaces"
	"github.com/containerd/containerd/v2/pkg/oci"
	"github.com/containerd/containerd/v2/pkg/shim"
	"golang.org/x/sys/unix"

	"github.com/containerd/containerd/v2/pkg/shutdown"
	"github.com/containerd/containerd/v2/plugins"
	"github.com/containerd/log"
	"github.com/containerd/plugin"
	"github.com/containerd/plugin/registry"

	katashim "github.com/kata-containers/kata-containers/src/runtime/pkg/containerd-shim-v2"
	katatypes "github.com/kata-containers/kata-containers/src/runtime/pkg/types"
)

func init() {
	registry.Register(&plugin.Registration{
		Type: plugins.TTRPCPlugin,
		ID:   "task",
		Requires: []plugin.Type{
			plugins.EventPlugin,
			plugins.InternalPlugin,
		},
		InitFn: func(ic *plugin.InitContext) (interface{}, error) {
			pp, err := ic.GetByID(plugins.EventPlugin, "publisher")
			if err != nil {
				return nil, err
			}
			ss, err := ic.GetByID(plugins.InternalPlugin, "shutdown")
			if err != nil {
				return nil, err
			}
			return newTaskService(ic.Context, pp.(shim.Publisher), ss.(shutdown.Service))
		},
	})
}

type manager struct {
	name string
}

func NewManager(name string) shim.Manager {
	return &manager{name: name}
}

func (m *manager) Info(_ context.Context, _ io.Reader) (*types.RuntimeInfo, error) {
	info := &types.RuntimeInfo{
		Name: m.Name(),
		Version: &types.RuntimeVersion{
			Version: "v0.0.1",
		},
	}
	return info, nil
}

func (m *manager) Name() string {
	return m.name
}

func (*manager) Start(ctx context.Context, id string, opts shim.StartOpts) (params shim.BootstrapParams, retErr error) {
	defer func() {
		if retErr != nil {
			log.G(ctx).WithField("error", retErr).Error("Start error")
		} else {
			log.G(ctx).Info("Start success")
		}
	}()
	params.Version = 3
	params.Protocol = "ttrpc"

	cmd, err := newCommand(ctx, id, opts.Address, opts.Debug)
	if err != nil {
		return params, err
	}

	address, err := shim.SocketAddress(ctx, opts.Address, id, false)
	if err != nil {
		return params, err
	}

	socket, err := shim.NewSocket(address)
	if err != nil {
		if !shim.SocketEaddrinuse(err) {
			return params, fmt.Errorf("create new shim socket: %w", err)
		}
		if shim.CanConnect(address) {
			params.Address = address
			return params, nil
		}
		if err := shim.RemoveSocket(address); err != nil {
			return params, fmt.Errorf("remove pre-existing socket: %w", err)
		}
		if socket, err = shim.NewSocket(address); err != nil {
			return params, fmt.Errorf("try create new shim socket 2x: %w", err)
		}
	}
	defer func() {
		if retErr != nil {
			_ = socket.Close()
			_ = shim.RemoveSocket(address)
		}
	}()

	f, err := socket.File()
	if err != nil {
		return params, err
	}

	cmd.ExtraFiles = append(cmd.ExtraFiles, f)

	if err := cmd.Start(); err != nil {
		_ = f.Close()
		return params, err
	}

	defer func() {
		if retErr != nil {
			_ = cmd.Process.Kill()
		}
	}()
	go func() {
		_ = cmd.Wait()
	}()

	params.Address = address
	return params, nil
}

const unmountFlags = unix.MNT_FORCE

func (*manager) Stop(ctx context.Context, id string) (stat shim.StopStatus, retErr error) {
	defer func() {
		if retErr != nil {
			log.G(ctx).WithField("error", retErr).Error("Stop error")
		} else {
			log.G(ctx).Info("Stop success")
		}
	}()
	cwd, err := os.Getwd()
	if err != nil {
		retErr = err
		return stat, err
	}

	bundlePath := filepath.Join(filepath.Dir(cwd), id)

	spec, err := oci.ReadSpec(path.Join(bundlePath, oci.ConfigFilename))
	if err == nil {
		if err = mount.UnmountRecursive(spec.Root.Path, unmountFlags); err != nil {
			log.G(ctx).WithError(err).Warn("failed to cleanup rootfs mount")
		}
	}

	return shim.StopStatus{
		ExitedAt: time.Now(),
		// TODO
	}, nil
}

func newCommand(ctx context.Context, id, containerdAddress string, debug bool) (*exec.Cmd, error) {
	ns, err := namespaces.NamespaceRequired(ctx)
	if err != nil {
		return nil, err
	}

	self, err := os.Executable()
	if err != nil {
		return nil, err
	}

	args := []string{
		"-namespace", ns,
		"-id", id,
		"-address", containerdAddress,
	}

	if debug {
		args = append(args, "-debug")
	}

	cmd := exec.Command(self, args...)

	return cmd, nil
}

func newTaskService(ctx context.Context, publisher shim.Publisher, sd shutdown.Service) (taskAPI.TTRPCTaskService, error) {
	// The shim.Publisher and shutdown.Service are usually useful for your task service,
	// but we don't need them in the exampleTaskService.
	return katashim.New(ctx, katatypes.DefaultKataRuntimeName, publisher, sd)
}

// var (
// 	_ = shim.TTRPCService(&exampleTaskService{})
// )

// type exampleTaskService struct {
// }

// // RegisterTTRPC allows TTRPC services to be registered with the underlying server
// func (s *exampleTaskService) RegisterTTRPC(server *ttrpc.Server) error {
// 	taskAPI.RegisterTaskService(server, s)
// 	return nil
// }

// // Create a new container
// func (s *exampleTaskService) Create(ctx context.Context, r *taskAPI.CreateTaskRequest) (_ *taskAPI.CreateTaskResponse, err error) {
// 	return nil, errdefs.ErrNotImplemented
// }

// // Start the primary user process inside the container
// func (s *exampleTaskService) Start(ctx context.Context, r *taskAPI.StartRequest) (*taskAPI.StartResponse, error) {
// 	return nil, errdefs.ErrNotImplemented
// }

// // Delete a process or container
// func (s *exampleTaskService) Delete(ctx context.Context, r *taskAPI.DeleteRequest) (*taskAPI.DeleteResponse, error) {
// 	return nil, errdefs.ErrNotImplemented
// }

// // Exec an additional process inside the container
// func (s *exampleTaskService) Exec(ctx context.Context, r *taskAPI.ExecProcessRequest) (*ptypes.Empty, error) {
// 	return nil, errdefs.ErrNotImplemented
// }

// // ResizePty of a process
// func (s *exampleTaskService) ResizePty(ctx context.Context, r *taskAPI.ResizePtyRequest) (*ptypes.Empty, error) {
// 	return nil, errdefs.ErrNotImplemented
// }

// // State returns runtime state of a process
// func (s *exampleTaskService) State(ctx context.Context, r *taskAPI.StateRequest) (*taskAPI.StateResponse, error) {
// 	return nil, errdefs.ErrNotImplemented
// }

// // Pause the container
// func (s *exampleTaskService) Pause(ctx context.Context, r *taskAPI.PauseRequest) (*ptypes.Empty, error) {
// 	return nil, errdefs.ErrNotImplemented
// }

// // Resume the container
// func (s *exampleTaskService) Resume(ctx context.Context, r *taskAPI.ResumeRequest) (*ptypes.Empty, error) {
// 	return nil, errdefs.ErrNotImplemented
// }

// // Kill a process
// func (s *exampleTaskService) Kill(ctx context.Context, r *taskAPI.KillRequest) (*ptypes.Empty, error) {
// 	return nil, errdefs.ErrNotImplemented
// }

// // Pids returns all pids inside the container
// func (s *exampleTaskService) Pids(ctx context.Context, r *taskAPI.PidsRequest) (*taskAPI.PidsResponse, error) {
// 	return nil, errdefs.ErrNotImplemented
// }

// // CloseIO of a process
// func (s *exampleTaskService) CloseIO(ctx context.Context, r *taskAPI.CloseIORequest) (*ptypes.Empty, error) {
// 	return nil, errdefs.ErrNotImplemented
// }

// // Checkpoint the container
// func (s *exampleTaskService) Checkpoint(ctx context.Context, r *taskAPI.CheckpointTaskRequest) (*ptypes.Empty, error) {
// 	return nil, errdefs.ErrNotImplemented
// }

// // Connect returns shim information of the underlying service
// func (s *exampleTaskService) Connect(ctx context.Context, r *taskAPI.ConnectRequest) (*taskAPI.ConnectResponse, error) {
// 	return nil, errdefs.ErrNotImplemented
// }

// // Shutdown is called after the underlying resources of the shim are cleaned up and the service can be stopped
// func (s *exampleTaskService) Shutdown(ctx context.Context, r *taskAPI.ShutdownRequest) (*ptypes.Empty, error) {
// 	os.Exit(0)
// 	return &ptypes.Empty{}, nil
// }

// // Stats returns container level system stats for a container and its processes
// func (s *exampleTaskService) Stats(ctx context.Context, r *taskAPI.StatsRequest) (*taskAPI.StatsResponse, error) {
// 	return nil, errdefs.ErrNotImplemented
// }

// // Update the live container
// func (s *exampleTaskService) Update(ctx context.Context, r *taskAPI.UpdateTaskRequest) (*ptypes.Empty, error) {
// 	return nil, errdefs.ErrNotImplemented
// }

// // Wait for a process to exit
// func (s *exampleTaskService) Wait(ctx context.Context, r *taskAPI.WaitRequest) (*taskAPI.WaitResponse, error) {
// 	return nil, errdefs.ErrNotImplemented
// }
