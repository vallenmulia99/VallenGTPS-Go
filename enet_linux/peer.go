package enet

// #include <enet/enet.h>
import "C"
import (
	"encoding/binary"
	"fmt"
	"math"
	"unsafe"
)

// EnetPeerState represents the state of a peer
type EnetPeerState int

// EnetPeerState constants
const (
	Disconnected EnetPeerState = iota
	Connecting
	AcknowledgingConnect
	ConnectionPending
	ConnectionSucceeded
	Connected
	DisconnectLater
	Disconnecting
	AcknowledgingDisconnect
	Zombie
)

// Peer is a peer which data packets may be sent or received from
type Peer interface {
	GetAddress() Address

	Disconnect(data uint32)
	DisconnectNow(data uint32)
	DisconnectLater(data uint32)

	SendBytes(data []byte, channel uint8, flags PacketFlags) error
	SendString(str string, channel uint8, flags PacketFlags) error
	SendPacket(packet Packet, channel uint8) error

	// SetData sets an arbitrary value against a peer. This is useful to attach some
	// application-specific data for future use, such as an identifier.
	//
	// http://enet.bespin.org/structENetPeer.html#a1873959810db7ac7a02da90469ee384e
	//
	// Note that due to the way the enet library works, if using this you are
	// responsible for clearing this data when the peer is finished with.
	// SetData(nil) will free underlying memory and avoid any leaks.
	//
	// See http://enet.bespin.org/Tutorial.html#ManageHost for an example of this
	// in the underlying library.
	SetData(data []byte)

	// GetData returns an application-specific value that's been set
	// against this peer. This returns nil if no data has been set.
	//
	// http://enet.bespin.org/structENetPeer.html#a1873959810db7ac7a02da90469ee384e
	GetData() []byte
	PeerTimeout(timeoutLimit, timeoutMinimum, timeoutMaximum uint32)
	GetConnectID() uint32
	State() EnetPeerState
}

// enetPeer is an implementation of the Peer interface
type enetPeer struct {
	cPeer *C.struct__ENetPeer
}

// NewPeer creates a new peer from a C peer
func (peer enetPeer) State() EnetPeerState {
	switch peer.cPeer.state {
	case C.ENET_PEER_STATE_DISCONNECTED:
		return Disconnected
	case C.ENET_PEER_STATE_CONNECTING:
		return Connecting
	case C.ENET_PEER_STATE_ACKNOWLEDGING_CONNECT:
		return AcknowledgingConnect
	case C.ENET_PEER_STATE_CONNECTION_PENDING:
		return ConnectionPending
	case C.ENET_PEER_STATE_CONNECTION_SUCCEEDED:
		return ConnectionSucceeded
	case C.ENET_PEER_STATE_CONNECTED:
		return Connected
	case C.ENET_PEER_STATE_DISCONNECT_LATER:
		return DisconnectLater
	case C.ENET_PEER_STATE_DISCONNECTING:
		return Disconnecting
	case C.ENET_PEER_STATE_ACKNOWLEDGING_DISCONNECT:
		return AcknowledgingDisconnect
	case C.ENET_PEER_STATE_ZOMBIE:
		return Zombie
	default:
		// Handle unexpected states
		return Disconnected // or another appropriate default
	}
}

// GetConnectID returns the connect ID of a peer
func (peer enetPeer) GetConnectID() uint32 {
	return uint32(peer.cPeer.connectID)
}

// GetAddress returns the address of a peer
func (peer enetPeer) GetAddress() Address {
	return &enetAddress{
		cAddr: peer.cPeer.address,
	}
}

// Disconnect a peer from a host
func (peer enetPeer) Disconnect(data uint32) {
	C.enet_peer_disconnect(
		peer.cPeer,
		(C.enet_uint32)(data),
	)
}

// DisconnectNow immediately disconnects a peer from a host
func (peer enetPeer) DisconnectNow(data uint32) {
	C.enet_peer_disconnect_now(
		peer.cPeer,
		(C.enet_uint32)(data),
	)
}

// DisconnectLater schedules a peer for disconnection
func (peer enetPeer) DisconnectLater(data uint32) {
	C.enet_peer_disconnect_later(
		peer.cPeer,
		(C.enet_uint32)(data),
	)
}

// PeerTimeout sets the timeout parameters for a peer
func (peer enetPeer) PeerTimeout(timeoutLimit, timeoutMin, timeoutMax uint32) {
	C.enet_peer_timeout(
		peer.cPeer,
		(C.enet_uint32)(timeoutLimit),
		(C.enet_uint32)(timeoutMin),
		(C.enet_uint32)(timeoutMax),
	)
}

// SendBytes sends a byte slice to a peer
func (peer enetPeer) SendBytes(data []byte, channel uint8, flags PacketFlags) error {
	packet, err := NewPacket(data, flags)
	if err != nil {
		return err
	}
	return peer.SendPacket(packet, channel)
}

// SendString sends a string to a peer
func (peer enetPeer) SendString(str string, channel uint8, flags PacketFlags) error {
	packet, err := NewPacket([]byte(str), flags)
	if err != nil {
		return err
	}
	return peer.SendPacket(packet, channel)
}

// SendPacket sends a packet to a peer
func (peer enetPeer) SendPacket(packet Packet, channel uint8) error {
	C.enet_peer_send(
		peer.cPeer,
		(C.enet_uint8)(channel),
		packet.(enetPacket).cPacket,
	)
	return nil
}

// SetData sets an arbitrary value against a peer. This is useful to attach some
func (peer enetPeer) SetData(data []byte) {
	if len(data) > math.MaxUint32 {
		panic(fmt.Sprintf("maximum peer data length is uint32 (%d)", math.MaxUint32))
	}

	// Free any data that was previously stored against this peer.
	existing := unsafe.Pointer(peer.cPeer.data)
	if existing != nil {
		C.free(existing)
	}

	// If nil, set this explicitly.
	if data == nil {
		peer.cPeer.data = nil
		return
	}

	// First 4 bytes stores how many bytes we have. This is so we can C.GoBytes when
	// retrieving which requires a byte length to read.
	b := make([]byte, len(data)+4)
	binary.LittleEndian.PutUint32(b, uint32(len(data)))
	// Join this header + data in to a contiguous slice
	copy(b[4:], data)
	// And write it out to C memory, storing our pointer.
	peer.cPeer.data = unsafe.Pointer(C.CBytes(b))
}

// GetData returns an application-specific value that's been set
func (peer enetPeer) GetData() []byte {
	ptr := unsafe.Pointer(peer.cPeer.data)

	if ptr == nil {
		return nil
	}

	// First 4 bytes are the bytes length.
	header := []byte{
		*(*byte)(unsafe.Add(ptr, 0)),
		*(*byte)(unsafe.Add(ptr, 1)),
		*(*byte)(unsafe.Add(ptr, 2)),
		*(*byte)(unsafe.Add(ptr, 3)),
	}

	return []byte(C.GoBytes(
		// Take from the start of the data.
		unsafe.Add(ptr, 4),
		// As many bytes as were indicated in the header.
		C.int(binary.LittleEndian.Uint32(header)),
	))
}
