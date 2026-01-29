package features

const (
	ConstructorCheck Feature = 1 << iota
	StructMethodCheck
)

type Feature uint8

func (c Feature) IsEnabled(other Feature) bool {
	return c&other != 0
}
