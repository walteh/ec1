package virtio

type VirtioRootfs struct {
	ImagePath string `json:"imagePath"`
}

func (v *VirtioRootfs) isVirtioDevice() {}

func VirtioRootfsNew(imagePath string) (*VirtioRootfs, error) {
	return &VirtioRootfs{
		ImagePath: imagePath,
	}, nil
}

// func (v *VirtioRootfs) toVZ() ([]string, error) {
// 	nvme, err := NVMExpressControllerNew(v.ImagePath)
// 	if err != nil {
// 		return nil, err
// 	}

// 	return nvme.ToCmdLine()
// }

// type VirtioFileHandleDeviceAttachment struct {
// 	File       *os.File
// 	DeviceType string
// }

// func (v *VirtioFileHandleDeviceAttachment) isVirtioDevice() {}

// func VirtioFileHandleDeviceAttachmentNew(file *os.File, deviceType string) (*VirtioFileHandleDeviceAttachment, error) {
// 	return &VirtioFileHandleDeviceAttachment{
// 		File:       file,
// 		DeviceType: deviceType,
// 	}, nil
// }

// func (v *VirtioFileHandleDeviceAttachment) toVZ() (vz.FileHandleDeviceAttachment, error) {
// 	yep, err := vz.NewFileHa(v.File, vz.DeviceSerial)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return yep, nil
// }
