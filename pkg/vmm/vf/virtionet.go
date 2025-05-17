package vf

import (
	"net"

	"github.com/Code-Hex/vz/v3"
	"gitlab.com/tozd/go/errors"

	log "github.com/sirupsen/logrus"

	"github.com/walteh/ec1/pkg/machines/virtio"
)

type VirtioNet struct {
	*virtio.VirtioNet
	localAddr *net.UnixAddr
}

func (dev *VirtioNet) toVz() (*vz.VirtioNetworkDeviceConfiguration, error) {
	var (
		mac *vz.MACAddress
		err error
	)

	if len(dev.MacAddress) == 0 {
		mac, err = vz.NewRandomLocallyAdministeredMACAddress()
	} else {
		mac, err = vz.NewMACAddress(dev.MacAddress)
	}
	if err != nil {
		return nil, errors.Errorf("creating mac address: %w", err)
	}
	var attachment vz.NetworkDeviceAttachment
	if dev.Socket != nil {
		attachment, err = vz.NewFileHandleNetworkDeviceAttachment(dev.Socket)
		if err != nil {
			return nil, errors.Errorf("creating file handle network attachment: %w", err)
		}
	} else {
		attachment, err = vz.NewNATNetworkDeviceAttachment()
		if err != nil {
			return nil, errors.Errorf("creating nat network attachment: %w", err)
		}
	}

	networkConfig, err := vz.NewVirtioNetworkDeviceConfiguration(attachment)
	if err != nil {
		return nil, errors.Errorf("creating network config: %w", err)
	}
	networkConfig.SetMACAddress(mac)

	return networkConfig, nil
}

func (dev *VirtioNet) AddToVirtualMachineConfig(vmConfig *vzVitualMachineConfigurationWrapper) error {
	log.Infof("Adding virtio-net device (nat: %t macAddress: [%s])", dev.Nat, dev.MacAddress)
	// if dev.Socket != nil {
	// 	log.Infof("Using fd %d", dev.Socket.Fd())
	// }
	// if dev.UnixSocketPath != "" {
	// 	log.Infof("Using unix socket %s", dev.UnixSocketPath)
	// 	if err := dev.connectUnixPath(); err != nil {
	// 		return err
	// 	}
	// }
	netConfig, err := dev.toVz()
	if err != nil {
		return errors.Errorf("converting virtio-net to vz: %w", err)
	}

	vmConfig.networkDevicesConfiguration = append(vmConfig.networkDevicesConfiguration, netConfig)

	return nil
}
