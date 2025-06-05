package toci

import (
	"io"
	"io/fs"
)

type OCIFile interface {
	fs.File
}

var registry = map[string][]byte{}

func MustRegisterImage(imageName string, fileName string, data fs.FS) {
	fle, err := data.Open(fileName)
	if err != nil {
		panic("opening file: " + err.Error())
	}
	registry[imageName], err = io.ReadAll(fle)
	if err != nil {
		panic("reading file: " + err.Error())
	}
}

func GetImage(imageName string) []byte {
	return registry[imageName]
}

func Registry() map[string][]byte {
	copy := make(map[string][]byte)
	for k, v := range registry {
		copy[k] = v
	}
	return copy
}
