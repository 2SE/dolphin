package security

import (
	"crypto/sha1"
	"encoding/base32"
	"encoding/binary"
	"encoding/hex"
	"math/rand"
	"net"
	"strings"
	"sync/atomic"
	"time"

	"golang.org/x/crypto/pbkdf2"
)

// ID represents a process-wide unique ID.
type ID uint64

// next is the next identifier. We seed it with the time in seconds
// to avoid collisions of ids between process restarts.
var next = uint64(
	time.Now().Sub(time.Date(2016, 5, 8, 1, 40, 0, 0, time.UTC)).Seconds(),
)

// NewID generates a new, process-wide unique ID.
func NewID() ID {
	return ID(atomic.AddUint64(&next, 1))
}

// Unique generates unique id based on the current id with a prefix and salt.
func (id ID) Unique(prefix uint64, salt string) string {
	buffer := [16]byte{}
	binary.BigEndian.PutUint64(buffer[:8], prefix)
	binary.BigEndian.PutUint64(buffer[8:], uint64(id))

	enc := pbkdf2.Key(buffer[:], []byte(salt), 4096, 16, sha1.New)
	return strings.Trim(base32.StdEncoding.EncodeToString(enc), "=")
}

// String converts the ID to a string representation.
func (id ID) String() string {
	buf := make([]byte, 10) // This will never be more than 9 bytes.
	l := binary.PutUvarint(buf, uint64(id))
	return strings.ToUpper(hex.EncodeToString(buf[:l]))
}

var hardware = initHardware()

// GetHardware retrieves the hardware address fingerprint which is based
// on the address of the first interface encountered.
func GetHardware() Fingerprint {
	return Fingerprint(hardware)
}

// Initializes the fingerprint
func initHardware() uint64 {
	var hardwareAddr [6]byte
	interfaces, err := net.Interfaces()
	if err == nil {
		for _, iface := range interfaces {
			if len(iface.HardwareAddr) >= 6 {
				copy(hardwareAddr[:], iface.HardwareAddr)
				return encode(hardwareAddr[:])
			}
		}
	}

	safeRandom(hardwareAddr[:])
	hardwareAddr[0] |= 0x01
	return encode(hardwareAddr[:])
}

func encode(mac net.HardwareAddr) (r uint64) {
	for _, b := range mac {
		r <<= 8
		r |= uint64(b)
	}
	return
}

func safeRandom(dest []byte) {
	if _, err := rand.Read(dest); err != nil {
		panic(err)
	}
}

// Fingerprint represents hardware fingerprint
type Fingerprint uint64

// String encodes PeerName as a string.
func (f Fingerprint) String() string {
	return intmac(uint64(f)).String()
}

// Hex returns the string in hex format.
func (f Fingerprint) Hex() string {
	return strings.Replace(f.String(), ":", "", -1)
}

// Converts int to hardware address
func intmac(key uint64) (r net.HardwareAddr) {
	r = make([]byte, 6)
	for i := 5; i >= 0; i-- {
		r[i] = byte(key)
		key >>= 8
	}
	return
}
