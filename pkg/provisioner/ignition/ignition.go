package ignition

import types_exp "github.com/coreos/ignition/v2/config/v3_6_experimental/types"

func NewIgnitionBootConfigProvider(cfg *types_exp.Config) *IgnitionBootConfigProvider {
	return &IgnitionBootConfigProvider{cfg: cfg}
}
