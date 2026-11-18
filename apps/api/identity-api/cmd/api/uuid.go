package main

import (
	"crypto/rand"
	"encoding/hex"
)

// uuidGenerator produces RFC 4122 version 4 UUIDs from crypto/rand. It uses only
// the standard library, so no identifier dependency is introduced.
type uuidGenerator struct{}

// newUUIDGenerator returns the default identifier generator.
func newUUIDGenerator() uuidGenerator { return uuidGenerator{} }

// New returns a random version 4 UUID string with the correct version and
// variant bits set.
func (uuidGenerator) New() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		// crypto/rand failing is unrecoverable; panic is appropriate for a
		// system-level entropy failure.
		panic("identity: crypto/rand unavailable: " + err.Error())
	}
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // variant 10

	buf := make([]byte, 36)
	hex.Encode(buf[0:8], b[0:4])
	buf[8] = '-'
	hex.Encode(buf[9:13], b[4:6])
	buf[13] = '-'
	hex.Encode(buf[14:18], b[6:8])
	buf[18] = '-'
	hex.Encode(buf[19:23], b[8:10])
	buf[23] = '-'
	hex.Encode(buf[24:36], b[10:16])
	return string(buf)
}
