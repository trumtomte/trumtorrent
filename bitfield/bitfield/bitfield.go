package bitfield

// Bitfields represents a simple Bit field implementation
type Bitfield []byte

func (b Bitfield) HasPiece(piece int) bool {
	index := piece / 8

	if index < 0 || index >= len(b) {
		return false
	}

	offset := 7 - piece%8
	return (b[index] >> uint(offset) & 1) == 1
}

func (b Bitfield) SetPiece(piece int) {
	index := piece / 8

	if index < 0 || index >= len(b) {
		return
	}

	offset := 7 - piece%8
	b[index] |= 1 << uint(offset)
}
