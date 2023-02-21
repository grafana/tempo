package network

import (
	"errors"
	"net"

	"github.com/DataDog/gohai/utils"
)

// Network holds network metadata about the host
type Network struct {
	// IpAddress is the ipv4 address for the host
	IpAddress string
	// IpAddressv6 is the ipv6 address for the host
	IpAddressv6 string
	// MacAddress is the macaddress for the host
	MacAddress string

	// TODO: the collect method also returns metadata about interfaces. They should be added to this struct.
	// Since it would require even more cleanup we'll do it in another PR when needed.
}

const name = "network"

func (self *Network) Name() string {
	return name
}

func (self *Network) Collect() (result interface{}, err error) {
	result, err = getNetworkInfo()
	if err != nil {
		return
	}

	interfaces, err := getMultiNetworkInfo()
	if err == nil && len(interfaces) > 0 {
		interfaceMap, ok := result.(map[string]interface{})
		if !ok {
			return
		}
		interfaceMap["interfaces"] = interfaces
	}
	return
}

// Get returns a Network struct already initialized, a list of warnings and an error. The method will try to collect as much
// metadata as possible, an error is returned if nothing could be collected. The list of warnings contains errors if
// some metadata could not be collected.
func Get() (*Network, []string, error) {
	networkInfo, err := getNetworkInfo()
	if err != nil {
		return nil, nil, err
	}

	return &Network{
		IpAddress:   utils.GetStringInterface(networkInfo, "ipaddress"),
		IpAddressv6: utils.GetStringInterface(networkInfo, "ipaddressv6"),
		MacAddress:  utils.GetStringInterface(networkInfo, "macaddress"),
	}, nil, nil
}

func getMultiNetworkInfo() (multiNetworkInfo []map[string]interface{}, err error) {
	ifaces, err := net.Interfaces()

	if err != nil {
		return multiNetworkInfo, err
	}
	for _, iface := range ifaces {
		_iface := make(map[string]interface{})
		_ipv4 := []string{}
		_ipv6 := []string{}
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			// interface down or loopback interface
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			// skip this interface but try the next
			continue
		}
		for _, addr := range addrs {
			ip, network, _ := net.ParseCIDR(addr.String())
			if ip == nil || ip.IsLoopback() {
				continue
			}
			if ip.To4() == nil {
				_ipv6 = append(_ipv6, ip.String())
				_iface["ipv6-network"] = network.String()
			} else {
				_ipv4 = append(_ipv4, ip.String())
				_iface["ipv4-network"] = network.String()
			}
			if len(iface.HardwareAddr.String()) > 0 {
				_iface["macaddress"] = iface.HardwareAddr.String()
			}
		}
		_iface["ipv4"] = _ipv4
		_iface["ipv6"] = _ipv6
		if len(_iface) > 0 {
			_iface["name"] = iface.Name
			multiNetworkInfo = append(multiNetworkInfo, _iface)
		}
	}
	return multiNetworkInfo, err
}

type Ipv6Address struct{}

func externalIpv6Address() (string, error) {
	ifaces, err := net.Interfaces()

	if err != nil {
		return "", err
	}

	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			// interface down or loopback interface
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			return "", err
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil || ip.IsLoopback() {
				continue
			}
			if ip.To4() != nil {
				// ipv4 address
				continue
			}
			return ip.String(), nil
		}
	}

	// We don't return an error if no IPv6 interface has been found. Indeed,
	// some orgs just don't have IPv6 enabled. If there's a network error, it
	// will pop out when getting the Mac address and/or the IPv4 address
	// (before this function's call; see network.go -> getNetworkInfo())
	return "", nil
}

type IpAddress struct{}

func externalIpAddress() (string, error) {
	ifaces, err := net.Interfaces()

	if err != nil {
		return "", err
	}

	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			// interface down or loopback interface
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			return "", err
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil || ip.IsLoopback() {
				continue
			}
			ip = ip.To4()
			if ip == nil {
				// not an ipv4 address
				continue
			}
			return ip.String(), nil
		}
	}
	return "", errors.New("not connected to the network")
}

type MacAddress struct{}

func macAddress() (string, error) {
	ifaces, err := net.Interfaces()

	if err != nil {
		return "", err
	}

	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			// interface down or loopback interface
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			return "", err
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil || ip.IsLoopback() || ip.To4() == nil {
				continue
			}
			return iface.HardwareAddr.String(), nil
		}
	}
	return "", errors.New("not connected to the network")
}
