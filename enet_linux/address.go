package enet

import (
	"unsafe"
)

// #include <enet/enet.h>
import "C"

// Address specifies a portable internet address structure.
type Address interface {
	// SetHostAny()
	BuildAny(addressType ENetAddressType)

	SetHost(addressType ENetAddressType, ip string)
	SetPort(port uint16)

	String() string
	GetPort() uint16
}

// enetAddress is the internal implementation of Address
type enetAddress struct {
	cAddr C.struct__ENetAddress
}

// enetAddressType is the internal implementation of Address
type ENetAddressType C.ENetAddressType

const (
	ENET_ADDRESS_TYPE_ANY  ENetAddressType = ENetAddressType(C.ENET_ADDRESS_TYPE_ANY)
	ENET_ADDRESS_TYPE_IPV4 ENetAddressType = ENetAddressType(C.ENET_ADDRESS_TYPE_IPV4)
	ENET_ADDRESS_TYPE_IPV6 ENetAddressType = ENetAddressType(C.ENET_ADDRESS_TYPE_IPV6)
)

// cAddress returns the C address of the address
func (addr *enetAddress) cAddress() *C.struct__ENetAddress {
	return &addr.cAddr
}

// BuildAny builds an address that can be used to bind to any host
func (addr *enetAddress) BuildAny(addressType ENetAddressType) {
	C.enet_address_build_any(&addr.cAddr, C.ENetAddressType(addressType))
}

/* SetHostAny sets the host of the address to ENET_HOST_ANY
func (addr *enetAddress) SetHostAny() {
	addr.cAddr.host = C.ENET_HOST_ANY
}*/

// SetHost sets the host of the address
func (addr *enetAddress) SetHost(addressType ENetAddressType, hostname string) {
	cHostname := C.CString(hostname)
	C.enet_address_set_host(
		&addr.cAddr,
		C.ENetAddressType(addressType),
		cHostname,
	)
	C.free(unsafe.Pointer(cHostname))
}

// SetPort sets the port number of the address
func (addr *enetAddress) SetPort(port uint16) {
	addr.cAddr.port = (C.enet_uint16)(port)
}

// String returns the IP address of the address
func (addr *enetAddress) String() string {
	buffer := C.malloc(32)
	C.enet_address_get_host_ip(
		&addr.cAddr,
		(*C.char)(buffer),
		32,
	)
	ret := C.GoString((*C.char)(buffer))
	C.free(buffer)
	return ret
}

// GetPort returns the port number of the address
func (addr *enetAddress) GetPort() uint16 {
	return uint16(addr.cAddr.port)
}

// NewAddress creates a new address
func NewAddress(addressType ENetAddressType, ip string, port uint16) Address {
	ret := enetAddress{}
	ret.SetHost(addressType, ip)
	ret.SetPort(port)
	return &ret
}

// NewListenAddress makes a new address ready for listening on ENET_HOST_ANY
func NewListenAddress(addressType ENetAddressType, port uint16) Address {
	ret := enetAddress{}
	// ret.BuildAny(addressType)
	if addressType == ENET_ADDRESS_TYPE_IPV4 {
		ret.SetHost(addressType, "0.0.0.0")
	} else {
		ret.SetHost(addressType, "::")
	}
	ret.SetPort(port)
	return &ret
}
