package testdata

import (
	"embed"
)

//go:embed *
var testdata embed.FS

func Testdata() *embed.FS {
	return &testdata
}
