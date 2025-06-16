package pkg

import (
	"errors"
	"io"
	"net"
)

type Packet struct {
	ID   int32
	Data []byte
}

func (p *Packet) Write(c net.Conn) error {
	// Write packet length
	length := int32(len(p.Data)) + int32(varIntLen(p.ID))
	err := writeVarInt(c, length)
	if err != nil {
		return err
	}

	// Write packet ID
	err = writeVarInt(c, p.ID)
	if err != nil {
		return err
	}

	// Write packet data if present
	if p.Data != nil {
		_, err = c.Write(p.Data)
	}
	return err
}

func ReadPacket(c net.Conn) (*Packet, error) {
	// Read packet length
	length, err := readVarInt(c)
	if err != nil {
		return nil, err
	}
	if length == 0 {
		return nil, errors.New("invalid packet length")
	}

	// Read packet ID
	id, err := readVarInt(c)
	if err != nil {
		return nil, err
	}

	// Read packet data
	buf := make([]byte, length-uint32(varIntLen(int32(id))))
	_, err = io.ReadFull(c, buf)
	if err != nil {
		return nil, err
	}

	return &Packet{
		ID:   int32(id),
		Data: buf,
	}, nil
}

func writeVarInt(w io.Writer, value int32) error {
	buf := make([]byte, varIntLen(value))
	n := varIntToBytes(buf, value)
	_, err := w.Write(buf[:n])
	return err
}

func writeUInt16(w io.Writer, value uint16) error {
	_, err := w.Write([]byte{byte(value >> 8), byte(value)})
	return err
}

func writeString(w io.Writer, value string) error {
	err := writeVarInt(w, int32(len(value)))
	if err != nil {
		return err
	}
	_, err = w.Write([]byte(value))
	return err
}

func readVarInt(r io.Reader) (uint32, error) {
	var result uint32
	var shift uint
	for range 5 {
		b, err := readByte(r)
		if err != nil {
			return 0, err
		}
		result |= uint32(b&0x7F) << shift
		if b&0x80 == 0 {
			return result, nil
		}
		shift += 7
	}
	return 0, errors.New("VarInt is too big")
}

func varIntLen(value int32) int {
	for i := 1; i < 5; i++ {
		if value < 1<<(7*i) {
			return i
		}
	}
	return 5
}

func varIntToBytes(buf []byte, num int32) int {
	i := 0
	for {
		temp := num & 0x7F
		num >>= 7
		if num != 0 {
			temp |= 0x80
		}
		buf[i] = byte(temp)
		i++
		if num == 0 {
			break
		}
	}
	return i
}

func readByte(r io.Reader) (byte, error) {
	var buf [1]byte
	_, err := r.Read(buf[:])
	return buf[0], err
}
