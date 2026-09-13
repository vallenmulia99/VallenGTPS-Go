package variant

import (
	"encoding/binary"
	"math"
)

const (
	TypeUnused    = 0
	TypeFloat     = 1
	TypeString    = 2
	TypeVector2   = 3
	TypeVector3   = 4
	TypeUint32    = 5
	TypeEntity    = 6
	TypeComponent = 7
	TypeRect      = 8
	TypeInt32     = 9
)

type Vec2f struct {
	X, Y float32
}

type Vec3f struct {
	X, Y, Z float32
}

type Variant struct {
	NetID int32
	Delay int32
	Args  []interface{}
}

func New(function string, args ...interface{}) *Variant {
	allArgs := append([]interface{}{function}, args...)
	return &Variant{
		NetID: -1,
		Delay: 0,
		Args:  allArgs,
	}
}

func NewWithNetID(function string, netID int32, args ...interface{}) *Variant {
	allArgs := append([]interface{}{function}, args...)
	return &Variant{
		NetID: netID,
		Delay: 0,
		Args:  allArgs,
	}
}

func (v *Variant) Pack() []byte {
	payload := v.SerializePayload()

	pkt := make([]byte, 60)

	binary.LittleEndian.PutUint32(pkt[0:], 4)

	binary.LittleEndian.PutUint32(pkt[4:], 1)

	binary.LittleEndian.PutUint32(pkt[8:], uint32(v.NetID))

	binary.LittleEndian.PutUint32(pkt[16:], 8)

	binary.LittleEndian.PutUint32(pkt[24:], uint32(v.Delay))

	binary.LittleEndian.PutUint32(pkt[56:], uint32(len(payload)))

	return append(pkt, payload...)
}

func (v *Variant) SerializePayload() []byte {
	buf := []byte{}

	buf = append(buf, byte(len(v.Args)))

	for i, arg := range v.Args {
		idx := byte(i)

		switch val := arg.(type) {
		case string:
			buf = append(buf, idx, TypeString)
			strBytes := []byte(val)
			lenBytes := make([]byte, 4)
			binary.LittleEndian.PutUint32(lenBytes, uint32(len(strBytes)))
			buf = append(buf, lenBytes...)
			buf = append(buf, strBytes...)

		case int:
			buf = append(buf, idx, TypeInt32)
			b := make([]byte, 4)
			binary.LittleEndian.PutUint32(b, uint32(val))
			buf = append(buf, b...)

		case int32:
			buf = append(buf, idx, TypeInt32)
			b := make([]byte, 4)
			binary.LittleEndian.PutUint32(b, uint32(val))
			buf = append(buf, b...)

		case uint32:
			buf = append(buf, idx, TypeUint32)
			b := make([]byte, 4)
			binary.LittleEndian.PutUint32(b, val)
			buf = append(buf, b...)

		case float32:
			buf = append(buf, idx, TypeFloat)
			b := make([]byte, 4)
			binary.LittleEndian.PutUint32(b, math.Float32bits(val))
			buf = append(buf, b...)

		case Vec2f:
			buf = append(buf, idx, TypeVector2)
			b := make([]byte, 8)
			binary.LittleEndian.PutUint32(b[0:], math.Float32bits(val.X))
			binary.LittleEndian.PutUint32(b[4:], math.Float32bits(val.Y))
			buf = append(buf, b...)

		case Vec3f:
			buf = append(buf, idx, TypeVector3)
			b := make([]byte, 12)
			binary.LittleEndian.PutUint32(b[0:], math.Float32bits(val.X))
			binary.LittleEndian.PutUint32(b[4:], math.Float32bits(val.Y))
			binary.LittleEndian.PutUint32(b[8:], math.Float32bits(val.Z))
			buf = append(buf, b...)

		default:
			buf = append(buf, idx, TypeInt32)
			b := make([]byte, 4)
			buf = append(buf, b...)
		}
	}

	return buf
}

func SendAction(action, str string) []byte {
	fmtAction := "action|" + action + "\n"
	text := fmtAction + str

	data := make([]byte, 4+len(text))
	data[0] = 3 
	copy(data[4:], text)
	return data
}

// SetBux creates the authentic OnSetBux variant packet matching Growtopia/GrowTavern protocol
func SetBux(gems int, isSupporter int) *Variant {
	return New("OnSetBux", int32(gems), int32(0), int32(isSupporter))
}
