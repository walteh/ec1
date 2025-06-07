package containerd

import (
	"github.com/containerd/containerd/v2/pkg/shim"
	"github.com/containerd/containerd/v2/pkg/shutdown"
	"github.com/containerd/containerd/v2/plugins"
	"github.com/containerd/plugin"
	"github.com/containerd/plugin/registry"
	"gitlab.com/tozd/go/errors"
)

func init() {
	RegisterPlugins()
}

func RegisterPlugins() {
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

			s, err := NewTaskService(ic.Context, pp.(shim.Publisher), ss.(shutdown.Service))
			if err != nil {
				return nil, errors.Errorf("failed to create task service: %w", err)
			}

			return WrapTaskServiceWithErrorLogging(s), nil
		},
	})
}
