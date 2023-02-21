//go:build android
// +build android

package platform

func (self *Platform) Collect() (interface{}, error) {
	return nil, nil
}

func Get() (*Platform, []string, error) {
	return nil, nil, nil
}
