package version

import (
	"fmt"
	"github.com/coreos/go-semver/semver"
)

var (
	Version    = "2.4.0+git"
	APIVersion = "unknown"
)

func init() {
	ver, err := semver.NewVersion(Version)
	if err == nil {
		APIVersion = fmt.Sprintf("%d.%d", ver.Major, ver.Minor)
	}
}
