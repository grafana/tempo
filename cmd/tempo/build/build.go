package build

import (
	"github.com/prometheus/common/version"
	prom "github.com/prometheus/prometheus/web/api/v1"
)

func GetVersion() prom.PrometheusVersion {
	return prom.PrometheusVersion{
		Version:   version.Version,
		Revision:  version.Revision,
		Branch:    version.Branch,
		BuildUser: version.BuildUser,
		BuildDate: version.BuildDate,
		GoVersion: version.GoVersion,
	}
}
