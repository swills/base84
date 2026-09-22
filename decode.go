package base84

import "fmt"

// Decode returns the bytes represented by a canonical Base84 string.
func Decode(encoded string) ([]byte, error) {
	decoded, err := StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, err
	}

	return decoded, nil
}

// DecodeString returns the bytes represented by a canonical Base84 string.
// On invalid input it returns the successfully decoded prefix and the error.
func (encoding *Encoding) DecodeString(encoded string) ([]byte, error) {
	decoded := make([]byte, encoding.DecodedLen(len(encoded)))
	written, err := encoding.Decode(decoded, []byte(encoded))

	return decoded[:written], err
}

// AppendDecode appends bytes represented by canonical Base84 data to destination.
// On invalid input it returns destination with the successfully decoded prefix and the error.
func (encoding *Encoding) AppendDecode(destination, encoded []byte) ([]byte, error) {
	prefixLength := len(destination)
	destination = append(destination, make([]byte, encoding.DecodedLen(len(encoded)))...)
	written, err := encoding.Decode(destination[prefixLength:], encoded)

	return destination[:prefixLength+written], err
}

// Decode writes bytes represented by canonical Base84 data to destination.
// On error it returns the bytes written before the invalid input or short destination.
func (encoding *Encoding) Decode(destination, encoded []byte) (int, error) {
	writer := byteWriter{destination: destination, written: 0, full: false}

	err := encoding.decodeTo(&writer, encoded)
	if err != nil {
		return writer.written, err
	}

	return writer.written, nil
}

func (encoding *Encoding) decodeTo(writer *byteWriter, encoded []byte) error {
	var (
		accumulator uint64
		pendingBits uint
		finalBits   uint
	)

	fullLength := len(encoded) / groupCharacters * groupCharacters
	for offset := 0; offset < fullLength; offset += groupCharacters {
		value, err := encoding.readChunk(encoded[offset:offset+groupCharacters], offset)
		if err != nil {
			return err
		}

		finalBits = groupBitCount(value)
		accumulator |= value << pendingBits
		pendingBits += finalBits

		for pendingBits >= 8 {
			byteCount := int(pendingBits / 8)
			if byteCount == 3 && writer.write3(accumulator) ||
				byteCount == 4 && writer.write4(accumulator) {
				accumulator >>= byteCount * 8
				pendingBits -= uint(byteCount * 8)

				continue
			}

			if !writer.writeByte(byte(accumulator & 0xff)) {
				return ErrNoSpaceLeft
			}

			accumulator >>= 8
			pendingBits -= 8
		}
	}

	if fullLength < len(encoded) {
		return encoding.decodeTail(writer, encoded[fullLength:], fullLength, accumulator, pendingBits)
	}

	return validateFinalGroup(encoded, accumulator, finalBits, pendingBits)
}

func (encoding *Encoding) decodeTail(
	writer *byteWriter,
	encoded []byte,
	offset int,
	accumulator uint64,
	pendingBits uint,
) error {
	value, err := encoding.readChunk(encoded, offset)
	if err != nil {
		return err
	}

	finalBits, err := tailBitCount(len(encoded), pendingBits, value)
	if err != nil {
		return fmt.Errorf("tail at byte %d: %w", offset, err)
	}

	accumulator |= value << pendingBits
	pendingBits += finalBits

	return writeDecodedBytes(writer, &accumulator, &pendingBits)
}

func writeDecodedBytes(writer *byteWriter, accumulator *uint64, pendingBits *uint) error {
	for *pendingBits >= 8 {
		byteCount := int(*pendingBits / 8)
		if byteCount == 3 && writer.write3(*accumulator) ||
			byteCount == 4 && writer.write4(*accumulator) {
			*accumulator >>= byteCount * 8
			*pendingBits -= uint(byteCount * 8)

			continue
		}

		if !writer.writeByte(byte(*accumulator & 0xff)) {
			return ErrNoSpaceLeft
		}

		*accumulator >>= 8
		*pendingBits -= 8
	}

	return nil
}

func validateFinalGroup(encoded []byte, accumulator uint64, finalBits, pendingBits uint) error {
	if len(encoded) > 0 && len(encoded)%groupCharacters == 0 &&
		(accumulator != 0 || finalBits-pendingBits <= 25) {
		return fmt.Errorf("final group: %w", ErrInvalidPadding)
	}

	return nil
}

func (writer *byteWriter) write3(value uint64) bool {
	if len(writer.destination)-writer.written < 3 {
		return false
	}

	output := writer.destination[writer.written:]
	output[0] = byte(value & 0xff)
	output[1] = byte((value >> 8) & 0xff)
	output[2] = byte((value >> 16) & 0xff)
	writer.written += 3

	return true
}

func (writer *byteWriter) write4(value uint64) bool {
	if len(writer.destination)-writer.written < 4 {
		return false
	}

	output := writer.destination[writer.written:]
	output[0] = byte(value & 0xff)
	output[1] = byte((value >> 8) & 0xff)
	output[2] = byte((value >> 16) & 0xff)
	output[3] = byte((value >> 24) & 0xff)
	writer.written += 4

	return true
}

func (encoding *Encoding) readChunk(chunk []byte, offset int) (uint64, error) {
	if len(chunk) == groupCharacters {
		_ = chunk[4]

		digit0 := encoding.inverse[chunk[0]]
		digit1 := encoding.inverse[chunk[1]]
		digit2 := encoding.inverse[chunk[2]]
		digit3 := encoding.inverse[chunk[3]]
		digit4 := encoding.inverse[chunk[4]]

		if digit0|digit1|digit2|digit3|digit4 != invalidDigit {
			value := uint32(digit0) + uint32(digit1)*84 + uint32(digit2)*7056 +
				uint32(digit3)*592704 + uint32(digit4)*49787136

			return uint64(value), nil
		}
	}

	var (
		value uint64
		place = uint64(1)
	)

	for index, character := range chunk {
		digit, valid := encoding.alphabetDigit(character)
		if !valid {
			return 0, fmt.Errorf(
				"character %q at byte %d: %w",
				character,
				offset+index,
				ErrInvalidCharacter,
			)
		}

		value += digit * place
		place *= alphabetSize
	}

	return value, nil
}

func (encoding *Encoding) alphabetDigit(character byte) (uint64, bool) {
	if encoding.inverse[character] == invalidDigit {
		return 0, false
	}

	return uint64(encoding.inverse[character]), true
}
