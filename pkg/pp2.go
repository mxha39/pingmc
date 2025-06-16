package pkg

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"net"
)

const (
	TCPv4 TransportProtocol = '\x11'
	TCPv6 TransportProtocol = '\x21'
	PROXY byte              = '\x21'
)

var (
	lengthV4      = uint16(12)
	lengthV6      = uint16(36)
	lengthV4Bytes = func() []byte {
		a := make([]byte, 2)
		binary.BigEndian.PutUint16(a, lengthV4)
		return a
	}()
	lengthV6Bytes = func() []byte {
		a := make([]byte, 2)
		binary.BigEndian.PutUint16(a, lengthV6)
		return a
	}()
	errUint16Overflow = errors.New("proxyproto: uint16 overflow")
	ErrInvalidAddress = errors.New("proxyproto: invalid address")
	SIGV2             = []byte{'\x0D', '\x0A', '\x0D', '\x0A', '\x00', '\x0D', '\x0A', '\x51', '\x55', '\x49', '\x54', '\x0A'}
)

type (
	TransportProtocol byte
	Header            struct {
		TransportProtocol TransportProtocol
		SourceAddr        net.Addr
		DestinationAddr   net.Addr
		rawTLVs           []byte
	}
)

func (header *Header) WriteTo(w io.Writer) (int64, error) {
	if header.TransportProtocol == 0 {
		header.TransportProtocol = TCPv4

		if IsIPv6(removePort(header.SourceAddr.String())) || IsIPv6(removePort(header.DestinationAddr.String())) {
			header.TransportProtocol = TCPv6
		}
	}

	buf, err := header.Format()
	if err != nil {
		return 0, err
	}
	_, err = w.Write(buf)
	return int64(len(buf)), err
}

func (ap TransportProtocol) IsIPv4() bool {
	return ap&0xF0 == 0x10
}

func (ap TransportProtocol) IsIPv6() bool {
	return ap&0xF0 == 0x20
}

func (ap TransportProtocol) IsStream() bool {
	return ap&0x0F == 0x01
}

func (ap TransportProtocol) toByte() byte {
	if ap.IsIPv4() && ap.IsStream() {
		return byte(TCPv4)
	}
	return byte(TCPv6)

}

func addTLVLen(cur []byte, tlvLen int) ([]byte, error) {
	if tlvLen == 0 {
		return cur, nil
	}
	curLen := binary.BigEndian.Uint16(cur)
	newLen := int(curLen) + tlvLen
	if newLen >= 1<<16 {
		return nil, errUint16Overflow
	}
	a := make([]byte, 2)
	binary.BigEndian.PutUint16(a, uint16(newLen))
	return a, nil
}

func (header *Header) Format() ([]byte, error) {
	var buf bytes.Buffer
	buf.Write(SIGV2)
	buf.WriteByte(PROXY)
	buf.WriteByte(header.TransportProtocol.toByte())
	var addrSrc, addrDst []byte
	if header.TransportProtocol.IsIPv4() {
		hdrLen, err := addTLVLen(lengthV4Bytes, len(header.rawTLVs))
		if err != nil {
			return nil, err
		}
		buf.Write(hdrLen)
		sourceIP, destIP, _ := header.IPs()
		addrSrc = sourceIP.To4()
		addrDst = destIP.To4()
	} else if header.TransportProtocol.IsIPv6() {
		hdrLen, err := addTLVLen(lengthV6Bytes, len(header.rawTLVs))
		if err != nil {
			return nil, err
		}
		buf.Write(hdrLen)
		sourceIP, destIP, _ := header.IPs()
		addrSrc = sourceIP.To16()
		addrDst = destIP.To16()
	}

	if addrSrc == nil || addrDst == nil {
		return nil, ErrInvalidAddress
	}
	buf.Write(addrSrc)
	buf.Write(addrDst)

	if sourcePort, destPort, ok := header.Ports(); ok {
		portBytes := make([]byte, 2)

		binary.BigEndian.PutUint16(portBytes, uint16(sourcePort))
		buf.Write(portBytes)

		binary.BigEndian.PutUint16(portBytes, uint16(destPort))
		buf.Write(portBytes)
	}

	if len(header.rawTLVs) > 0 {
		buf.Write(header.rawTLVs)
	}

	return buf.Bytes(), nil
}

func (header *Header) Ports() (sourcePort, destPort int, ok bool) {
	if sourceAddr, destAddr, ok := header.TCPAddrs(); ok {
		return sourceAddr.Port, destAddr.Port, true
	} else {
		return 0, 0, false
	}
}

func (header *Header) IPs() (sourceIP, destIP net.IP, ok bool) {
	if sourceAddr, destAddr, ok := header.TCPAddrs(); ok {
		return sourceAddr.IP, destAddr.IP, true
	} else {
		return nil, nil, false
	}
}

func (header *Header) TCPAddrs() (sourceAddr, destAddr *net.TCPAddr, ok bool) {
	if !header.TransportProtocol.IsStream() {
		return nil, nil, false
	}
	sourceAddr, sourceOK := header.SourceAddr.(*net.TCPAddr)
	destAddr, destOK := header.DestinationAddr.(*net.TCPAddr)
	return sourceAddr, destAddr, sourceOK && destOK
}

func IsIPv6(address string) bool {
	ip := net.ParseIP(address)
	return ip != nil && ip.To4() == nil
}
