package memory

// Memory holds memory metadata about the host
type Memory struct {
	// TotalBytes is the total memory for the host in byte
	TotalBytes uint64
	// SwapTotalBytes is the swap memory size in byte (Unix only)
	SwapTotalBytes uint64
}

const name = "memory"

func (self *Memory) Name() string {
	return name
}

func (self *Memory) Collect() (result interface{}, err error) {
	result, err = getMemoryInfo()
	return
}

// Get returns a Memory struct already initialized, a list of warnings and an error. The method will try to collect as much
// metadata as possible, an error is returned if nothing could be collected. The list of warnings contains errors if
// some metadata could not be collected.
func Get() (*Memory, []string, error) {
	// Legacy code from gohai returns memory in:
	// - byte for Windows
	// - mix of byte and MB for OSX
	// - KB on linux
	//
	// this method being new we can align this behavior to return bytes everywhere without breaking backward
	// compatibility

	mem, swap, warnings, err := getMemoryInfoByte()
	if err != nil {
		return nil, nil, err
	}

	return &Memory{
		TotalBytes:     mem,
		SwapTotalBytes: swap,
	}, warnings, nil
}
