package reliquary

func DecryptCommand(key []byte, encrypted []byte) []byte {

	if isTraceEnabled() {
		trace("before decryption data", "bytes", bytesAsHex(encrypted), "len", len(encrypted))
	}

	for i := 0; i < len(encrypted); i++ {
		encrypted[i] ^= key[i%len(key)]
	}

	if isTraceEnabled() {
		trace("after decryption data", "bytes", bytesAsHex(encrypted), "len", len(encrypted))
	}

	return encrypted
}
