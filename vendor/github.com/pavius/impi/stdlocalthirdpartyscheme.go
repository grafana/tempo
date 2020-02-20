package impi

type stdLocalThirdPartyScheme struct{}

// newStdLocalThirdPartyScheme returns a new stdLocalThirdPartyScheme
func newStdLocalThirdPartyScheme() *stdLocalThirdPartyScheme {
	return &stdLocalThirdPartyScheme{}
}

// getMaxNumGroups returns max number of groups the scheme allows
func (sltp *stdLocalThirdPartyScheme) getMaxNumGroups() int {
	return 3
}

// getMixedGroupsAllowed returns whether a group can contain imports of different types
func (sltp *stdLocalThirdPartyScheme) getMixedGroupsAllowed() bool {
	return false
}

// GetAllowedGroupOrders returns which group orders are allowed
func (sltp *stdLocalThirdPartyScheme) getAllowedImportOrders() [][]importType {
	return [][]importType{
		{importTypeStd},
		{importTypeLocal},
		{importTypeThirdParty},
		{importTypeStd, importTypeLocal},
		{importTypeStd, importTypeThirdParty},
		{importTypeLocal, importTypeThirdParty},
		{importTypeStd, importTypeLocal, importTypeThirdParty},
	}
}
