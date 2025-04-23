package testdata

import (
	"embed"
)

//go:embed *
var data embed.FS

func FS() *embed.FS {
	return &data
}
