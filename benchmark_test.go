package base84

import (
	"strconv"
	"testing"
)

var (
	benchmarkBytesSink  []byte
	benchmarkStringSink string
	benchmarkCountSink  int
)

var benchmarkPayloadSizes = [...]int{16, 31, 1024, 65536}

func benchmarkPayload(size int) []byte {
	payload := make([]byte, size)
	for index := range payload {
		payload[index] = byte((index*131 + 17) & 0xff)
	}

	return payload
}

func BenchmarkEncodeAllocate(b *testing.B) {
	for _, size := range benchmarkPayloadSizes {
		b.Run(strconv.Itoa(size), func(b *testing.B) {
			payload := benchmarkPayload(size)

			var encoded string

			b.ReportAllocs()
			b.SetBytes(int64(size))

			for b.Loop() {
				encoded = StdEncoding.EncodeToString(payload)
			}

			benchmarkStringSink = encoded
		})
	}
}

func BenchmarkEncodeReuse(b *testing.B) {
	for _, size := range benchmarkPayloadSizes {
		b.Run(strconv.Itoa(size), func(b *testing.B) {
			payload := benchmarkPayload(size)
			destination := make([]byte, StdEncoding.EncodedLen(size))

			var (
				written int
				err     error
			)

			b.ReportAllocs()
			b.SetBytes(int64(size))

			for b.Loop() {
				written, err = StdEncoding.Encode(destination, payload)
			}

			benchmarkBytesSink = destination[:written]
			benchmarkCountSink = written

			if err != nil {
				b.Fatal(err)
			}
		})
	}
}

func BenchmarkDecodeAllocate(b *testing.B) {
	for _, size := range benchmarkPayloadSizes {
		b.Run(strconv.Itoa(size), func(b *testing.B) {
			payload := benchmarkPayload(size)
			encoded := StdEncoding.EncodeToString(payload)

			var (
				decoded []byte
				err     error
			)

			b.ReportAllocs()
			b.SetBytes(int64(size))

			for b.Loop() {
				decoded, err = StdEncoding.DecodeString(encoded)
			}

			benchmarkBytesSink = decoded

			if err != nil {
				b.Fatal(err)
			}
		})
	}
}

func BenchmarkDecodeReuse(b *testing.B) {
	for _, size := range benchmarkPayloadSizes {
		b.Run(strconv.Itoa(size), func(b *testing.B) {
			payload := benchmarkPayload(size)
			encoded := []byte(StdEncoding.EncodeToString(payload))
			destination := make([]byte, StdEncoding.DecodedLen(len(encoded)))

			var (
				written int
				err     error
			)

			b.ReportAllocs()
			b.SetBytes(int64(size))

			for b.Loop() {
				written, err = StdEncoding.Decode(destination, encoded)
			}

			benchmarkBytesSink = destination[:written]
			benchmarkCountSink = written

			if err != nil {
				b.Fatal(err)
			}
		})
	}
}
