package enet

// #include <enet/enet.h>
import "C"
import (
	"encoding/binary"
	"errors"
	"unsafe"
)

// PacketFlags are bit constants
type PacketFlags uint32

const (
	// PacketFlagReliable packets must be received by the target peer and resend attempts
	// should be made until the packet is delivered
	PacketFlagReliable PacketFlags = C.ENET_PACKET_FLAG_RELIABLE

	// PacketFlagUnsequenced packets will not be sequenced with other packets not supported
	// for reliable packets
	PacketFlagUnsequenced = C.ENET_PACKET_FLAG_UNSEQUENCED

	// PacketFlagNoAllocate packets will not allocate data, and user must supply it instead
	PacketFlagNoAllocate = C.ENET_PACKET_FLAG_NO_ALLOCATE

	// PacketFlagUnreliableFragment packets will be fragmented using unreliable (instead of
	// reliable) sends if it exceeds the MTU
	PacketFlagUnreliableFragment = C.ENET_PACKET_FLAG_UNRELIABLE_FRAGMENT

	// PacketFlagSent specifies whether the packet has been sent from all queues it has been
	// entered into
	PacketFlagSent = C.ENET_PACKET_FLAG_SENT
)

// Packet may be sent to or received from a peer
type Packet interface {
	Destroy()
	GetData() []byte
	GetFlags() PacketFlags
}

// enetPacket is a wrapper around the C ENetPacket struct
type enetPacket struct {
	cPacket *C.struct__ENetPacket
}

// Destroy frees the memory associated with the packet
func (packet enetPacket) Destroy() {
	C.enet_packet_destroy(packet.cPacket)
}

// GetData returns the data associated with the packet
func (packet enetPacket) GetData() []byte {
	return C.GoBytes(
		unsafe.Pointer(packet.cPacket.data),
		(C.int)(packet.cPacket.dataLength),
	)
}

// GetFlags returns the flags associated with the packet
func (packet enetPacket) GetFlags() PacketFlags {
	return (PacketFlags)(packet.cPacket.flags)
}

// NewPacket creates a new packet to send to peers
func NewPacket(data []byte, flags PacketFlags) (Packet, error) {
	buffer := C.CBytes(data)
	packet := C.enet_packet_create(
		buffer,
		(C.size_t)(len(data)),
		(C.enet_uint32)(flags),
	)
	C.free(buffer)

	if packet == nil {
		return nil, errors.New("unable to create packet")
	}

	return enetPacket{
		cPacket: packet,
	}, nil
}

// GetMessageFromPacket returns the message from a packet
func GetMessageFromPacket(packet Packet) string {
	gamePacket := packet.GetData()
	copy(gamePacket[len(gamePacket)-1:], []byte{0})
	return string(gamePacket[4:])
}

// SendPacket sends a packet to a peer
func SendPacket(peer Peer, gameMessageType int32, strData string) error {
	packetSize := 5 + len(strData)
	netPacket := make([]byte, packetSize)

	binary.LittleEndian.PutUint32(netPacket[0:4], uint32(gameMessageType))
	if strData != "" {
		copy(netPacket[4:4+len(strData)], []byte(strData))
	}
	netPacket[4+len(strData)] = 0
	packet, err := NewPacket(netPacket, PacketFlagReliable)
	if err != nil {
		return errors.New("unable to create packet on SendPacket.")
	}
	peer.SendPacket(packet, 0)
	return nil
}

// SendRawPacket sends a raw packet to a peer
func SendRawPacket(peer Peer, gameMessageType int32, data []byte) error {
	packetSize := 5 + len(data)
	netPacket := make([]byte, packetSize)
	binary.LittleEndian.PutUint32(netPacket[0:4], uint32(gameMessageType))
	if data != nil {
		copy(netPacket[4:4+len(data)], data)
	}
	netPacket[4+len(data)] = 0
	packet, err := NewPacket(netPacket, PacketFlagReliable)
	if err != nil {
		return errors.New("unable to create packet on SendRawPacket.")
	}
	peer.SendPacket(packet, 0)
	return nil
}
