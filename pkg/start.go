package pkg

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/pflag"
)

const (
	seperator  = "──────────────────────────────────────────────────────"
	LabelColor = "\033[91m"
	Reset      = "\033[0m"
)

type (
	StatusResponse struct {
		Version struct {
			Name     string `json:"name"`
			Protocol int    `json:"protocol"`
		} `json:"version"`
		Players struct {
			Max    int            `json:"max"`
			Online int            `json:"online"`
			Sample []PlayerSample `json:"sample"`
		} `json:"players"`

		Description MessageWrapper `json:"description"`
		Favicon     string         `json:"favicon"`
	}

	PlayerSample struct {
		Name string `json:"name"`
		UUID string `json:"id"`
	}

	MessageWrapper struct {
		m Message
	}
)

func (cd *MessageWrapper) String() string {
	return cd.m.String()
}

func (cd *MessageWrapper) UnmarshalJSON(data []byte) error {
	if data[0] == byte('"') {
		return json.Unmarshal(data, &cd.m.Text)
	}
	return json.Unmarshal(data, &cd.m)
}

func Start() {
	log.SetFlags(0)
	log.SetPrefix("")

	pp2 := pflag.Bool("pp2", false, "Proxy Protocol v2")
	spoof := pflag.StringP("spoof", "s", "", "Spoofed IP")
	version := pflag.IntP("version", "v", 766, "Protocol Version")
	icon := pflag.StringP("saveIcon", "i", "", "Save Icon")
	showPlayers := pflag.Bool("show", false, "Show Players")
	useV4 := pflag.BoolP("useV4", "4", false, "Use IPv4")
	useV6 := pflag.BoolP("useV6", "6", false, "Use IPv6")
	pflag.Parse()

	args := pflag.Args()
	if len(args) < 1 {
		fmt.Println("Usage: ./pingmc <host> [options]")
		return
	}

	var (
		err  error
		port = 25565
		host = args[0]
	)

	host, err = normalizeHostPort(host, "25565")
	if err != nil {
		log.Fatal(err)
	}

	host, portStr, err := net.SplitHostPort(host)
	if err != nil {
		log.Fatal(err)
		return
	}
	port, err = strconv.Atoi(portStr)
	if err != nil {
		log.Fatal(err)
		return
	}

	// Resolve SRV record
	if port == 25565 {
		_, srvAddr, err := net.LookupSRV("minecraft", "tcp", host)
		if err == nil && len(srvAddr) > 0 {
			port = int(srvAddr[0].Port)
			host = srvAddr[0].Target[:len(srvAddr[0].Target)-1]
		}
	}

	proto := "tcp"
	if *useV4 {
		proto = "tcp4"
	}
	if *useV6 {
		proto = "tcp6"
	}

	conn, err := net.Dial(proto, net.JoinHostPort(host, strconv.Itoa(port)))
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	if *pp2 {
		dstAddr, err := net.ResolveTCPAddr(proto, net.JoinHostPort(host, strconv.Itoa(port)))
		if err != nil {
			log.Fatal(err)
		}

		header := Header{
			SourceAddr: &net.TCPAddr{
				IP:   net.ParseIP("127.0.0.1"),
				Port: 43932, // random port
			},
			DestinationAddr: dstAddr,
		}
		if _, err = (&header).WriteTo(conn); err != nil {
			log.Println("failed to write proxy protocol header:", err)
			return
		}
	}

	buf := bytes.NewBuffer(nil)
	writeVarInt(buf, int32(*version)) // Protocol version
	if *spoof != "" {
		writeString(buf, *spoof) // Spoofed hostname
	} else {
		writeString(buf, host) // Server address
	}
	writeUInt16(buf, uint16(port)) // Server port
	writeVarInt(buf, 1)            // Next state (1 for status)

	// Send c->s handshake packet (ID 0x00)
	err = (&Packet{
		ID:   0x00,
		Data: buf.Bytes(),
	}).Write(conn)
	if err != nil {
		log.Fatal("Failed to send handshake:", err)
	}

	// Send c->s status request (ID 0x00)
	err = (&Packet{
		ID:   0x00,
		Data: nil,
	}).Write(conn)
	if err != nil {
		log.Fatal("Failed to send status request:", err)
	}

	// Read s->c response
	packet, err := ReadPacket(conn)
	if err != nil {
		log.Fatal("Failed to read status response:", err)
	}

	var status StatusResponse
	err = json.Unmarshal(packet.Data[strings.Index(string(packet.Data), "{"):], &status)
	if err != nil {
		log.Fatal(err)
	}

	// Ping
	val := time.Now().UnixMilli()
	encodedTime := make([]byte, 8)
	binary.LittleEndian.PutUint64(encodedTime, uint64(val))
	err = (&Packet{
		ID:   0x01,
		Data: encodedTime,
	}).Write(conn)
	if err != nil {
		log.Fatal("Failed to send ping:", err)
	}

	// Read s->c response
	packet, err = ReadPacket(conn)
	if err != nil {
		log.Fatal("Failed to read response:", err)
	}
	if uint64(val) != binary.LittleEndian.Uint64(packet.Data) {
		log.Println("Invalid pong response")
	}
	ping := time.Now().UnixMilli() - val

	if icon != nil && *icon != "" {
		split := strings.Split(status.Favicon, ",")

		if len(split) > 1 {
			decodeString, err := base64.StdEncoding.DecodeString(split[1])
			if err != nil {
				log.Fatal(err)
			}

			decode, _, err := image.Decode(strings.NewReader(string(decodeString)))
			if err != nil {
				log.Fatal(err)
			}

			fileName := *icon
			if !strings.HasSuffix(fileName, ".png") {
				fileName += ".png"
			}
			file, err := os.Create(fileName)
			if err != nil {
				log.Fatal(err)
			}
			defer file.Close()

			err = png.Encode(file, decode)
			if err != nil {
				log.Fatal(err)
			}
		} else {
			log.Printf("icon not found for %s", host)
		}
	}

	log.Println(seperator)
	log.Printf("%sTarget: %s%s\n", LabelColor, Reset, host)
	if *spoof != "" {
		log.Printf("%sSpoof: %s%s\n", LabelColor, Reset, *spoof)
	}
	log.Printf("%sIP-Address: %s%s\n", LabelColor, Reset, removePort(conn.RemoteAddr().String()))
	if port != 25565 {
		log.Printf("%sPort: %s%d\n", LabelColor, Reset, port)
	}
	log.Printf("%sVersion:%s %s - %d\n", LabelColor, Reset, Text(status.Version.Name), status.Version.Protocol)
	log.Printf("%sPlayers: %s%d / %d\n", LabelColor, Reset, status.Players.Online, status.Players.Max)
	log.Printf("%sPing: %s%d ms\n", LabelColor, Reset, ping)
	log.Printf("\n%s\n", status.Description.String())
	if *showPlayers && len(status.Players.Sample) > 0 {
		log.Printf("Players: %s\n", formatSamples(status.Players.Sample))
	}
	log.Println(seperator)

}

func formatSamples(players []PlayerSample) string {
	var text string
	for _, v := range players {
		text += "\n"

		if strings.Contains(v.Name, "§") {
			text += Text(v.Name).String()
		} else {
			text += "- " + Text(v.Name).String() + " (" + v.UUID + ")"
		}
	}
	return text
}

func removePort(str string) string {
	host, _, err := net.SplitHostPort(str)
	if err != nil {
		return str
	}
	return host
}

func normalizeHostPort(input string, defaultPort string) (string, error) {
	host, port, err := net.SplitHostPort(input)
	if err == nil {
		p, err := strconv.Atoi(port)
		if err != nil || p < 0 || p > 65535 {
			return "", fmt.Errorf("invalid port: %s", port)
		}
		return net.JoinHostPort(host, port), nil
	}

	if strings.Count(input, ":") >= 2 {
		if !strings.HasPrefix(input, "[") {
			input = "[" + input + "]"
		}
		input += ":"
	}
	if !strings.Contains(input, ":") {
		input += ":"
	}

	return input + defaultPort, nil
}
