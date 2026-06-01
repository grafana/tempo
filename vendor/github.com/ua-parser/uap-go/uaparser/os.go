package uaparser

type Os struct {
	Family     string
	Major      string
	Minor      string
	Patch      string
	PatchMinor string `yaml:"patch_minor"`
}

func (osp *osParser) Match(line string, os *Os) {
	matches := osp.Reg.FindStringSubmatchIndex(line)
	if len(matches) > 0 {
		os.Family = string(osp.Reg.ExpandString(nil, osp.OSReplacement, line, matches))
		os.Major = string(osp.Reg.ExpandString(nil, osp.V1Replacement, line, matches))
		os.Minor = string(osp.Reg.ExpandString(nil, osp.V2Replacement, line, matches))
		os.Patch = string(osp.Reg.ExpandString(nil, osp.V3Replacement, line, matches))
		os.PatchMinor = string(osp.Reg.ExpandString(nil, osp.V4Replacement, line, matches))
	}
}

func (os *Os) ToString() string {
	var str string
	if os.Family != "" {
		str += os.Family
	}
	version := os.ToVersionString()
	if version != "" {
		str += " " + version
	}
	return str
}

func (os *Os) ToVersionString() string {
	var version string
	if os.Major != "" {
		version += os.Major
	}
	if os.Minor != "" {
		version += "." + os.Minor
	}
	if os.Patch != "" {
		version += "." + os.Patch
	}
	if os.PatchMinor != "" {
		version += "." + os.PatchMinor
	}
	return version
}
