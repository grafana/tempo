package platform

// Platform holds metadata about the host
type Platform struct {
	// GoVersion is the golang version.
	GoVersion string
	// GoOS is equal to "runtime.GOOS"
	GoOS string
	// GoArch is equal to "runtime.GOARCH"
	GoArch string

	// KernelName is the kernel name (ex:  "windows", "Linux", ...)
	KernelName string
	// KernelRelease the kernel release (ex: "10.0.20348", "4.15.0-1080-gcp", ...)
	KernelRelease string
	// Hostname is the hostname for the host
	Hostname string
	// Machine the architecture for the host (is: x86_64 vs arm).
	Machine string
	// OS is the os name description (ex: "GNU/Linux", "Windows Server 2022 Datacenter", ...)
	OS string

	// Family is the OS family (Windows only)
	Family string

	// KernelVersion the kernel version, Unix only
	KernelVersion string
	// Processor is the processor type, Unix only (ex "x86_64", "arm", ...)
	Processor string
	// HardwarePlatform is the hardware name, Linux only (ex "x86_64")
	HardwarePlatform string
}

const name = "platform"

func (self *Platform) Name() string {
	return name
}
