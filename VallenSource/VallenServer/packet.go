package server

import (
	"encoding/binary"
	"math"
)

// ─────────────────────────────────────────────
// GamePacket — Struktur paket game (60 byte header)
// ─────────────────────────────────────────────

// GamePacket merepresentasikan header 60 byte game packet ENet.
type GamePacket struct {
	PacketCreate int32
	Type         int32
	NetID        int32
	UID          int32
	State        int32
	Count        float32
	ID           int32
	PosX         float32
	PosY         float32
	SpeedX       float32
	SpeedY       float32
	Idk          float32
	PunchX       int32
	PunchY       int32
	Size         uint32
}

// parseGamePacket membaca GamePacket dari raw bytes (minimal 60 byte).
func parseGamePacket(data []byte) GamePacket {
	if len(data) < 60 {
		return GamePacket{}
	}
	return GamePacket{
		PacketCreate: int32(binary.LittleEndian.Uint32(data[0:])),
		Type:         int32(binary.LittleEndian.Uint32(data[4:])),
		NetID:        int32(binary.LittleEndian.Uint32(data[8:])),
		UID:          int32(binary.LittleEndian.Uint32(data[12:])),
		State:        int32(binary.LittleEndian.Uint32(data[16:])),
		Count:        math.Float32frombits(binary.LittleEndian.Uint32(data[20:])),
		ID:           int32(binary.LittleEndian.Uint32(data[24:])),
		PosX:         math.Float32frombits(binary.LittleEndian.Uint32(data[28:])),
		PosY:         math.Float32frombits(binary.LittleEndian.Uint32(data[32:])),
		SpeedX:       math.Float32frombits(binary.LittleEndian.Uint32(data[36:])),
		SpeedY:       math.Float32frombits(binary.LittleEndian.Uint32(data[40:])),
		Idk:          math.Float32frombits(binary.LittleEndian.Uint32(data[44:])),
		PunchX:       int32(binary.LittleEndian.Uint32(data[48:])),
		PunchY:       int32(binary.LittleEndian.Uint32(data[52:])),
		Size:         binary.LittleEndian.Uint32(data[56:]),
	}
}

// buildGamePacket menyusun game packet (60 byte header + optional extraData).
func buildGamePacket(gp GamePacket, extraData []byte) []byte {
	buf := make([]byte, 60)
	binary.LittleEndian.PutUint32(buf[0:], 4) // NET_MESSAGE_GAME_PACKET
	binary.LittleEndian.PutUint32(buf[4:], uint32(gp.Type))
	binary.LittleEndian.PutUint32(buf[8:], uint32(gp.NetID))
	binary.LittleEndian.PutUint32(buf[12:], uint32(gp.UID))
	binary.LittleEndian.PutUint32(buf[16:], uint32(gp.State))
	binary.LittleEndian.PutUint32(buf[20:], math.Float32bits(gp.Count))
	binary.LittleEndian.PutUint32(buf[24:], uint32(gp.ID))
	binary.LittleEndian.PutUint32(buf[28:], math.Float32bits(gp.PosX))
	binary.LittleEndian.PutUint32(buf[32:], math.Float32bits(gp.PosY))
	binary.LittleEndian.PutUint32(buf[36:], math.Float32bits(gp.SpeedX))
	binary.LittleEndian.PutUint32(buf[40:], math.Float32bits(gp.SpeedY))
	binary.LittleEndian.PutUint32(buf[44:], math.Float32bits(gp.Idk))
	binary.LittleEndian.PutUint32(buf[48:], uint32(gp.PunchX))
	binary.LittleEndian.PutUint32(buf[52:], uint32(gp.PunchY))
	binary.LittleEndian.PutUint32(buf[56:], uint32(len(extraData)))

	if len(extraData) > 0 {
		buf = append(buf, extraData...)
	}
	return buf
}
