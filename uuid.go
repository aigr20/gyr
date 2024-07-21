package gyr

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"
)

type UUID [16]byte

var (
	mxUUID sync.Mutex
	seq    = 0
)

// Generate a UUIDv7. Heavy inspiration taken from https://github.com/google/uuid for the implementation.
func NewUUID() UUID {
	mxUUID.Lock()
	defer mxUUID.Unlock()
	now := time.Now().UnixMilli()
	seq += 1

	var uuid UUID
	// 6 byte = 48 bit = timestamp in ms
	uuid[0] = byte(now >> 40) // 40 = 48 - 8, 48 is target bit length
	uuid[1] = byte(now >> 32)
	uuid[2] = byte(now >> 24)
	uuid[3] = byte(now >> 16)
	uuid[4] = byte(now >> 8)
	uuid[5] = byte(now)

	// 112 = 0b01110000, guarantees that first 4 bits (the version) are 0b0111 (7)
	uuid[6] = 112 | (15 & byte(seq>>8))
	uuid[7] = byte(seq)

	rand.Read(uuid[8:])
	uuid[8] = (uuid[8] & 63) | 128

	return uuid
}

func (uuid UUID) String() string {
	var out [36]byte

	out[8] = '-'
	out[13] = '-'
	out[18] = '-'
	out[23] = '-'
	hex.Encode(out[:], uuid[:4])
	hex.Encode(out[9:13], uuid[4:6])
	hex.Encode(out[14:18], uuid[6:8])
	hex.Encode(out[19:23], uuid[8:10])
	hex.Encode(out[24:], uuid[10:])

	return string(out[:])
}
