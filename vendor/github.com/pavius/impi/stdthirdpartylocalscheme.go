package impi

type stdThirdPartyLocalScheme struct{}

// newStdThirdPartyLocalScheme returns a new stdThirdPartyLocalScheme
func newStdThirdPartyLocalScheme() *stdThirdPartyLocalScheme {
	return &stdThirdPartyLocalScheme{}
}

// getMaxNumGroups returns max number of groups the scheme allows
func (sltp *stdThirdPartyLocalScheme) getMaxNumGroups() int {
	return 3
}

// getMixedGroupsAllowed returns whether a group can contain imports of different types
func (sltp *stdThirdPartyLocalScheme) getMixedGroupsAllowed() bool {
	return false
}

// GetAllowedGroupOrders returns which group orders are allowed
func (sltp *stdThirdPartyLocalScheme) getAllowedImportOrders() [][]importType {
	return [][]importType{
		{importTypeStd},
		{importTypeLocal},
		{importTypeThirdParty},
		{importTypeStd, importTypeLocal},
		{importTypeStd, importTypeThirdParty},
		{importTypeThirdParty, importTypeLocal},
		{importTypeStd, importTypeThirdParty, importTypeLocal},
	}
}
