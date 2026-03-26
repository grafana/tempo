package uaparser

type UserAgent struct {
	Family string
	Major  string
	Minor  string
	Patch  string
}

func (uap *uaParser) Match(line string, ua *UserAgent) {
	matches := uap.Reg.FindStringSubmatchIndex(line)
	if len(matches) > 0 {
		ua.Family = string(uap.Reg.ExpandString(nil, uap.FamilyReplacement, line, matches))
		ua.Major = string(uap.Reg.ExpandString(nil, uap.V1Replacement, line, matches))
		ua.Minor = string(uap.Reg.ExpandString(nil, uap.V2Replacement, line, matches))
		ua.Patch = string(uap.Reg.ExpandString(nil, uap.V3Replacement, line, matches))
	}
}

func (ua *UserAgent) ToString() string {
	var str string
	if ua.Family != "" {
		str += ua.Family
	}
	version := ua.ToVersionString()
	if version != "" {
		str += " " + version
	}
	return str
}

func (ua *UserAgent) ToVersionString() string {
	var version string
	if ua.Major != "" {
		version += ua.Major
	}
	if ua.Minor != "" {
		version += "." + ua.Minor
	}
	if ua.Patch != "" {
		version += "." + ua.Patch
	}
	return version
}
