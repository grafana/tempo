package uaparser

import "strings"

type Device struct {
	Family string
	Brand  string
	Model  string
}

func (dp *deviceParser) Match(line string, dvc *Device) {
	matches := dp.Reg.FindStringSubmatchIndex(line)

	if len(matches) == 0 {
		return
	}

	dvc.Family = string(dp.Reg.ExpandString(nil, dp.DeviceReplacement, line, matches))
	dvc.Family = strings.TrimSpace(dvc.Family)

	dvc.Brand = string(dp.Reg.ExpandString(nil, dp.BrandReplacement, line, matches))
	dvc.Brand = strings.TrimSpace(dvc.Brand)

	dvc.Model = string(dp.Reg.ExpandString(nil, dp.ModelReplacement, line, matches))
	dvc.Model = strings.TrimSpace(dvc.Model)
}

func (dvc *Device) ToString() string {
	return dvc.Family
}
