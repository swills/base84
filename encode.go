package base84

type byteWriter struct {
	destination []byte
	written     int
	full        bool
}

// Encode returns the canonical Base84 representation of source.
func Encode(source []byte) string {
	return StdEncoding.EncodeToString(source)
}

// EncodeToString returns the canonical Base84 representation of source.
func (encoding *Encoding) EncodeToString(source []byte) string {
	return string(encoding.AppendEncode(nil, source))
}

// AppendEncode appends the canonical Base84 representation of source to destination.
func (encoding *Encoding) AppendEncode(destination, source []byte) []byte {
	prefixLength := len(destination)
	destination = append(destination, make([]byte, encoding.EncodedLen(len(source)))...)
	writer := byteWriter{destination: destination[prefixLength:], written: 0, full: false}
	encoding.encodeTo(&writer, source)

	return destination[:prefixLength+writer.written]
}

// Encode writes the canonical Base84 representation of source to destination.
// If destination is too short, it returns the bytes written and ErrNoSpaceLeft.
func (encoding *Encoding) Encode(destination, source []byte) (int, error) {
	writer := byteWriter{destination: destination, written: 0, full: false}
	encoding.encodeTo(&writer, source)

	if writer.full {
		return writer.written, ErrNoSpaceLeft
	}

	return writer.written, nil
}

func (encoding *Encoding) encodeTo(writer *byteWriter, source []byte) {
	var (
		accumulator uint64
		bitCount    uint
	)

	for _, sourceByte := range source {
		accumulator |= uint64(sourceByte) << bitCount
		bitCount += 8

		if bitCount <= groupBits {
			continue
		}

		low := uint32(accumulator & lowWordMask)
		bits := groupBitCount(uint64(low))

		value := low
		if bits == groupBits {
			value &= uint32(groupMask)
		}

		if !encoding.writeGroup(writer, value) {
			encoding.writeDigits(writer, value, groupCharacters)
		}

		if writer.full {
			return
		}

		accumulator >>= bits
		bitCount -= bits
	}

	if bitCount == 0 {
		return
	}

	characters := tailCharacterCount(bitCount)
	if fitsOneCharacter(bitCount, accumulator) {
		characters = 1
	}

	encoding.writeDigits(writer, uint32(accumulator), characters)
}

func (encoding *Encoding) writeGroup(writer *byteWriter, value uint32) bool {
	if len(writer.destination)-writer.written < groupCharacters {
		return false
	}

	output := writer.destination[writer.written:]
	output[0] = encoding.forward[value%alphabetSize]
	value /= alphabetSize
	output[1] = encoding.forward[value%alphabetSize]
	value /= alphabetSize
	output[2] = encoding.forward[value%alphabetSize]
	value /= alphabetSize
	output[3] = encoding.forward[value%alphabetSize]
	value /= alphabetSize
	output[4] = encoding.forward[value%alphabetSize]
	writer.written += groupCharacters

	return true
}

func (encoding *Encoding) writeDigits(writer *byteWriter, value uint32, count int) {
	if len(writer.destination)-writer.written >= count {
		output := writer.destination[writer.written : writer.written+count]
		for index := range output {
			output[index] = encoding.forward[value%alphabetSize]
			value /= alphabetSize
		}

		writer.written += count

		return
	}

	for range count {
		if !writer.writeByte(encoding.forward[value%alphabetSize]) {
			return
		}

		value /= alphabetSize
	}
}

func (writer *byteWriter) writeByte(value byte) bool {
	if writer.written == len(writer.destination) {
		writer.full = true

		return false
	}

	writer.destination[writer.written] = value
	writer.written++

	return true
}
