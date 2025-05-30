package units

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"runtime"
	"strings"

	"gitlab.com/tozd/go/errors"
)

type Platform string

const (
	PlatformLinuxAMD64  Platform = "linux/amd64"
	PlatformLinuxARM64  Platform = "linux/arm64"
	PlatformDarwinAMD64 Platform = "darwin/amd64"
	PlatformDarwinARM64 Platform = "darwin/arm64"
)

const (
	OSLinux  = "linux"
	OSDarwin = "darwin"
)

const (
	// not supported yet
	ArchAMD64 = "amd64"
	ArchARM64 = "arm64"
)

func (p Platform) OS() string {
	return strings.Split(string(p), "/")[0]
}

func (p Platform) Arch() string {
	parts := strings.Split(string(p), "/")
	if len(parts) == 1 {
		return ""
	} else if len(parts) == 2 {
		return parts[1]
	} else {
		return strings.Join(parts[1:], "/")
	}
}

func (p Platform) String() string {
	return string(p)
}

func HostPlatform() Platform {
	return Platform(fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH))
}

// ParsePlatform parses a platform string and returns a Platform type
func ParsePlatform(platformStr string) (Platform, error) {
	if platformStr == "" {
		return "", errors.New("platform string cannot be empty")
	}

	// Check if it's a known variant first
	if variant, ok := supportedPlatformVariants[platformStr]; ok {
		return variant, nil
	}

	// Validate the format (should be "os/arch" or "os/arch/variant")
	parts := strings.Split(platformStr, "/")
	if len(parts) < 2 {
		return "", errors.Errorf("invalid platform format: %s (expected os/arch)", platformStr)
	}

	platform := Platform(platformStr)
	return platform, nil
}

func ResolvePlatformVariant(platform string) Platform {
	if variant, ok := supportedPlatformVariants[platform]; ok {
		return variant
	}
	return Platform(platform)
}

func (p Platform) IsSupported() bool {
	return supportedPlatforms[p]
}

var supportedPlatformVariants = map[string]Platform{
	"linux/arm/v8":   PlatformLinuxARM64,
	"linux/arm64/v8": PlatformLinuxARM64,
}

var supportedPlatforms = map[Platform]bool{
	PlatformLinuxAMD64:  true,
	PlatformLinuxARM64:  true,
	PlatformDarwinAMD64: true,
	PlatformDarwinARM64: true,
}

var supportedPlatformsHash string

func init() {
	hash := sha256.New()
	for platform := range supportedPlatforms {
		hash.Write([]byte(platform))
	}
	supportedPlatformsHash = hex.EncodeToString(hash.Sum(nil))
}

func SupportedPlatformsHash() string {
	return supportedPlatformsHash
}
