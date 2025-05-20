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

func (v *VirtioRootfs) toVZ() ([]string, error) {
	nvme, err := NVMExpressControllerNew(v.ImagePath)
	if err != nil {
		return nil, err
	}

	return nvme.ToCmdLine()
}
