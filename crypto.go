package reliquary

import (
	"encoding/binary"
	"github.com/goark/mt/mt19937"
)

type Key struct {
	_bytes []byte
}

func (k *Key) decryptCommand(encrypted []byte) []byte {

	if isTraceEnabled() {
		logger.Trace().
			Str("bytes", bytesAsHex(encrypted)).
			Int("len", len(encrypted)).
			Msg("data before decrypt")
	}

	for i := 0; i < len(encrypted); i++ {
		encrypted[i] ^= k._bytes[i%len(k._bytes)]
	}

	if isTraceEnabled() {
		logger.Trace().
			Str("bytes", bytesAsHex(encrypted)).
			Int("len", len(encrypted)).
			Msg("data after decrypt")
	}

	return encrypted
}

func newKeyBytesFromSeed(seed uint64) []byte {
	gen := mt19937.New((int64)(seed))

	key := make([]byte, 0)
	for i := 0; i < 512; i++ {
		n := gen.Uint64()
		key = binary.BigEndian.AppendUint64(key, n)
	}
	return key
}
