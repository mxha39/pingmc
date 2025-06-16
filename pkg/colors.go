package pkg

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// Source: https://github.com/Tnze/go-mc/blob/master/chat/message.go

const (
	Black       = "black"
	DarkBlue    = "dark_blue"
	DarkGreen   = "dark_green"
	DarkAqua    = "dark_aqua"
	DarkRed     = "dark_red"
	DarkPurple  = "dark_purple"
	Gold        = "gold"
	Gray        = "gray"
	DarkGray    = "dark_gray"
	Blue        = "blue"
	Green       = "green"
	Aqua        = "aqua"
	Red         = "red"
	LightPurple = "light_purple"
	Yellow      = "yellow"
	White       = "white"
)

type Message struct {
	Text string `json:"text" nbt:"text"`

	Bold          bool   `json:"bold,omitempty" nbt:"bold,omitempty"`
	Italic        bool   `json:"italic,omitempty" nbt:"italic,omitempty"`
	UnderLined    bool   `json:"underlined,omitempty" nbt:"underlined,omitempty"`
	StrikeThrough bool   `json:"strikethrough,omitempty" nbt:"strikethrough,omitempty"`
	Obfuscated    bool   `json:"obfuscated,omitempty" nbt:"obfuscated,omitempty"`
	Color         string `json:"color,omitempty" nbt:"color,omitempty"`

	With  []Message      `json:"with,omitempty" nbt:"with,omitempty"`
	Extra []MixedMessage `json:"extra,omitempty" nbt:"extra,omitempty"`
}
type MixedMessage struct {
	Message
	Raw string
}

func (m *MixedMessage) UnmarshalJSON(data []byte) error {
	var str string
	if err := json.Unmarshal(data, &str); err == nil {
		m.Text = str
		return nil
	}

	var msg Message
	if err := json.Unmarshal(data, &msg); err != nil {
		return fmt.Errorf("MixedMessage: unable to unmarshal %s", string(data))
	}
	m.Message = msg
	return nil
}

func Text(str string) Message {
	return Message{Text: str}
}

var fmtCode = map[byte]string{
	'0': "30",
	'1': "34",
	'2': "32",
	'3': "36",
	'4': "31",
	'5': "35",
	'6': "33",
	'7': "37",
	'8': "90",
	'9': "94",
	'a': "92",
	'b': "96",
	'c': "91",
	'd': "95",
	'e': "93",
	'f': "97",

	'k': "", //random
	'l': "1",
	'm': "9",
	'n': "4",
	'o': "3",
	'r': "0",
}

var colors = map[string]string{
	Black:       "30",
	DarkBlue:    "34",
	DarkGreen:   "32",
	DarkAqua:    "36",
	DarkRed:     "31",
	DarkPurple:  "35",
	Gold:        "33",
	Gray:        "37",
	DarkGray:    "90",
	Blue:        "94",
	Green:       "92",
	Aqua:        "96",
	Red:         "91",
	LightPurple: "95",
	Yellow:      "93",
	White:       "97",
}

// ClearString return the message String without escape sequence for ansi color.
func (m Message) ClearString() string {
	var msg strings.Builder
	text, _ := TransCtrlSeq(m.Text, false)
	msg.WriteString(text)

	// handle translate

	if m.Extra != nil {
		for i := range m.Extra {
			msg.WriteString(m.Extra[i].ClearString())
		}
	}
	return msg.String()
}

// String return the message string with escape sequence for ansi color.
// To convert Translated Message to string, you must set
// On Windows, you may want print this string using github.com/mattn/go-colorable.
func (m Message) String() string {
	var msg, format strings.Builder
	if m.Bold {
		format.WriteString("1;")
	}
	if m.Italic {
		format.WriteString("3;")
	}
	if m.UnderLined {
		format.WriteString("4;")
	}
	if m.StrikeThrough {
		format.WriteString("9;")
	}
	if m.Color != "" {
		format.WriteString(colors[m.Color] + ";")
	}
	if format.Len() > 0 {
		msg.WriteString("\033[" + format.String()[:format.Len()-1] + "m")
	}

	text, ok := TransCtrlSeq(m.Text, true)
	msg.WriteString(text)

	if m.Extra != nil {
		for i := range m.Extra {
			msg.WriteString(m.Extra[i].String())
		}
	}

	if format.Len() > 0 || ok {
		msg.WriteString("\033[0m")
	}
	return msg.String()
}

var fmtPat = regexp.MustCompile(`(?i)§[\dA-FK-OR]`)

// TransCtrlSeq will transform control sequences into ANSI code
// or simply filter them. Depends on the second argument.
// if the str contains control sequences, returned change=true.
func TransCtrlSeq(str string, ansi bool) (dst string, change bool) {
	dst = fmtPat.ReplaceAllStringFunc(
		str,
		func(str string) string {
			f, ok := fmtCode[str[2]]
			if ok {
				if ansi {
					change = true
					return "\033[" + f + "m" // enable, add ANSI code
				}
				return "" // disable, remove the § code
			}
			return str // not a § code
		},
	)
	return
}
