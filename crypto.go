package reliquary

func DecryptCommand(key []byte, encrypted []byte) []byte {
	for i := 0; i < len(encrypted); i++ {
		encrypted[i] ^= key[i%len(key)]
	}
	return encrypted
}
