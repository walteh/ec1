package vz

import (
	"fmt"
	"os"
	"plugin"
)

type VZPlugin struct {
	plugin *plugin.Plugin
}

func NewVZPlugin(file string) (*VZPlugin, error) {

	// make sure the file exists
	if _, err := os.Stat(file); os.IsNotExist(err) {
		return nil, fmt.Errorf("file does not exist: %s", file)
	}

	plugin, err := plugin.Open(file)
	if err != nil {
		return nil, err
	}

	return &VZPlugin{
		plugin: plugin,
	}, nil
}

func (v *VZPlugin) TestVZ(kernelPath string, log func(string, ...interface{})) error {

	log("Looking up TestVM\n")
	sym, err := v.plugin.Lookup("TestVM")
	if err != nil {
		return err
	}

	log("TestVM found\n")

	wrapper, ok := sym.(func(string, func(string, ...interface{})) error)
	if !ok {
		return fmt.Errorf("TestVZ is not a function")
	}

	log("calling TestVM\n")

	err = wrapper(kernelPath, log)
	if err != nil {
		return err
	}

	log("TestVM completed\n")

	return nil
}
