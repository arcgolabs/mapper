package main

import (
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/arcgolabs/mapper"
)

type Packet struct {
	Version uint16
	Kind    string
	Payload []byte
}

func (p *Packet) UnmarshalBinary(data []byte) error {
	if len(data) < 3 {
		return errors.New("packet too short")
	}

	p.Version = binary.BigEndian.Uint16(data[:2])
	p.Kind = string(data[2:3])
	p.Payload = append([]byte(nil), data[3:]...)
	return nil
}

func main() {
	raw := []byte{0x00, 0x01, 0x01, 0x66, 0x6f, 0x6f}
	packetFromBytes, err := mapper.Map[Packet](raw)
	fmt.Printf("from bytes: %#v err=%v\n", packetFromBytes, err)

	packetFromString, err := mapper.Map[Packet](string(raw))
	fmt.Printf("from string: %#v err=%v\n", packetFromString, err)

	_, err = mapper.Map[Packet]([]byte{0x01})
	fmt.Printf("from invalid bytes err=%v\n", err)
}
