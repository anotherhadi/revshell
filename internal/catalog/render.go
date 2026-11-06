package catalog

import (
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
)

// Vars are the values substituted into an Entry's {ip}/{port}/{shell}
// placeholders.
type Vars struct {
	IP    string
	Port  int
	Shell string
}

type Encoding int

const (
	NoEncoding Encoding = iota
	URLEncoding
	URLEncodingDouble
	Base64Encoding
)

// Render fills in e.Command's placeholders. For Shellcode entries, {ip} and
// {port} are hex-escaped (`\x..`) instead of substituted literally, the way
// a reverse_tcp payload expects its target baked into the instruction
// stream; enc is ignored in that case, matching the upstream UI.
func Render(e Entry, v Vars, enc Encoding) string {
	ip, port := v.IP, strconv.Itoa(v.Port)
	if e.Category == Shellcode {
		ip, port = toBytes(v.IP, v.Port)
	}

	out := strings.NewReplacer(
		"{ip}", ip,
		"{port}", port,
		"{shell}", v.Shell,
	).Replace(e.Command)

	if e.Category == Shellcode {
		return out
	}
	switch enc {
	case URLEncoding:
		out = percentEncode(out)
	case URLEncodingDouble:
		out = percentEncode(percentEncode(out))
	case Base64Encoding:
		out = base64.StdEncoding.EncodeToString([]byte(out))
	}
	return out
}

// toBytes mirrors the generator's own toBytes(): a dotted-quad IP becomes 4
// `\xHH` bytes and the port becomes 2 `\xHH` bytes (big-endian, as shellcode
// expects it in network byte order); anything else is left as `\x..`
// placeholders for the caller to patch by hand.
func toBytes(ip string, port int) (ipBytes, portBytes string) {
	octets := strings.Split(ip, ".")
	if len(octets) == 4 {
		var b strings.Builder
		ok := true
		for _, o := range octets {
			n, err := strconv.Atoi(o)
			if err != nil || n < 0 || n > 0xFF {
				ok = false
				break
			}
			fmt.Fprintf(&b, "\\x%02x", n)
		}
		if ok {
			ipBytes = b.String()
		}
	}
	if ipBytes == "" {
		ipBytes = "\\x..\\x..\\x..\\x.."
	}

	if port >= 0 && port <= 0xFFFF {
		portBytes = fmt.Sprintf("\\x%02x\\x%02x", port>>8, port&0xFF)
	} else {
		portBytes = "\\x..\\x.."
	}
	return ipBytes, portBytes
}

// percentEncode is JS's encodeURIComponent with !'()* also escaped, to be
// RFC 3986 safe (the generator's fixedEncodeURIComponent).
func percentEncode(s string) string {
	const unreserved = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_.~"
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		if strings.IndexByte(unreserved, c) >= 0 {
			b.WriteByte(c)
		} else {
			fmt.Fprintf(&b, "%%%02X", c)
		}
	}
	return b.String()
}
