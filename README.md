# base84

`base84` is a dependency-free Go package and command-line tool for Base84 encoding.
It implements the five-character format from
[jedisct1/zig-base84](https://github.com/jedisct1/zig-base84), where each full
group consumes either 31 or 32 input bits. That project is a library, not a
CLI, so this command-line tool does not claim CLI compatibility with it. For
format background, see the upstream repository and
[the Base84 format note](https://00f.net/2026/09/09/base84/).

## Package

Import `github.com/swills/base84` and encode bytes to a canonical Base84 string:

```go
encoded := base84.Encode([]byte("hello"))
decoded, err := base84.Decode(encoded)
if err != nil {
	panic(err)
}
```

`Decode` is strict. It rejects characters outside the Base84 alphabet and
non-canonical encodings, including invalid tail padding and full final groups
that do not encode a valid final byte count. Check the sentinel errors with
`errors.Is` because decode errors include their byte position or context:

```go
decoded, err := base84.Decode(encoded)
switch {
case errors.Is(err, base84.ErrInvalidCharacter):
	// The input contains a byte outside the Base84 alphabet.
case errors.Is(err, base84.ErrInvalidPadding):
	// The input is not a canonical Base84 representation.
case err != nil:
	panic(err)
}
```

`Decode` returns a newly allocated byte slice and never accepts ambiguous
padding forms.

### Custom encodings and buffers

`StdEncoding` and `StandardEncoding` initially reference the same standard
encoding. `NewEncoding` creates an encoding from exactly 84 distinct, non-NUL
ASCII bytes. Any ASCII byte except NUL is allowed, including CR and LF when
they are part of the supplied alphabet:

```go
alphabet := base84.Alphabet[1:] + base84.Alphabet[:1]
encoding, err := base84.NewEncoding(alphabet)
if err != nil {
	return fmt.Errorf("create Base84 encoding: %w", err)
}
encoded := encoding.EncodeToString([]byte("hello"))
```

Use `AppendEncode` and `AppendDecode` when the destination should grow. They
append to the supplied slice and return it, with `AppendDecode` also returning
strict decode errors. `EncodedLen` and `DecodedLen` return upper bounds, which
make them suitable for capacity planning rather than exact output lengths.

For fixed destinations, `Encoding.Encode` and `Encoding.Decode` return the
number of bytes written and `ErrNoSpaceLeft` when the destination is too short.
Either method may leave a partial output prefix in the destination. For normal
growable Go buffers, prefer `AppendEncode` and `AppendDecode`.

## Compatibility and testing

The test suite mirrors upstream fixed vectors and errors. It also exhausts all
byte inputs through length 2, all standard encoded strings through length 3,
50,000 deterministic round trips, 50,000 deterministic alteration attempts,
and size bound checks. This gives broad compatibility evidence, not a
mathematical proof for unbounded inputs.

## CLI

`base84` follows applicable command-line conventions from FreeBSD's `base64(1)`,
Fourmilab `base64`, and GNU coreutils `base64`. It does not claim exact
behavioral compatibility with those utilities.

Its invocation is:

```text
base84 [options] [input [output]]
```

The command encodes by default. `-e` and `--encode` select encoding explicitly;
`-d` and `--decode` select decoding. `-w` and `--wrap` set the encoded line
width, accepting either a separate value or an attached short value such as
`-w76`. The default is `0`, which disables wrapping.

When decoding, `-i` and `--ignore-garbage`, plus the Fourmilab-compatible
aliases `-n` and `--noerrcheck`, ignore bytes outside the Base84 alphabet.
Canonical Base84 and padding validation still apply after filtering. Without
these options, decoding ignores the six ASCII whitespace bytes: space, tab, CR,
LF, vertical tab, and form feed. Other invalid or non-canonical input is
rejected.

`-h`, `--help`, and the Fourmilab-compatible `-u` print help to standard output.
`--version` prints the version. Options may appear before, between, or after
operands until `--`. Boolean short options may be clustered, for example `-di`
and `-dn`.

Accept up to two operands: input followed by output. Either operand may be `-`
for the corresponding standard stream.

Nonempty encoded output ends with a newline. Empty input encodes to zero bytes.
Decoded output is raw binary bytes and never gains a newline. Usage errors use
status 2. GNU coreutils `base64` wraps at 76 columns by default, while `base84`
defaults to no wrapping.

Examples:

```sh
printf 'hello' | ./base84
./base84 --encode input.bin encoded.base84
./base84 --decode encoded.base84 output.bin
./base84 -w76 input.bin -
./base84 input.bin --decode -
./base84 -- -literal-name output.base84
printf '[@EtkbB\n' | ./base84 -d > output.bin
./base84 --version
```

## Development

Available Task targets:

```sh
task fmt
task lint
task test
task cover
task build
task bench
task bench-zig
task
```

`task` runs the default workflow: lint, test, and build. Run the same checks
directly with:

```sh
golangci-lint run
go test -race -shuffle=on -count=1 ./...
go build -o base84 ./cmd/base84
```

`task build` creates a stripped binary with `-s -w`. Its `VERSION` defaults to
`git describe`, with `devel` as the fallback. Override it with
`task build VERSION=v1.2.3`. Direct `go build` remains a valid unstripped
development build and does not inject Git metadata.

## Benchmarks

Run the Go suite with `task bench`. Run the Zig fixed-buffer comparison with
`task bench-zig`; it expects a sibling `../zig-base84` checkout by default.
Set `ZIG_BASE84_ROOT` when that checkout lives elsewhere:

```sh
ZIG_BASE84_ROOT=/path/to/zig-base84 task bench-zig
```

For comparable FreeBSD runs, pin each command to the same logical CPU:

```sh
cpuset -l 2 task bench
cpuset -l 2 task bench-zig
```

### Apple M1 Pro, macOS

Recorded 2026-09-19 with Go 1.27.1 and Zig 0.16.0. Go fixed-buffer
measurements combine two five-sample runs immediately before and after the Zig
suite. macOS does not provide the `cpuset` CPU pinning used for the FreeBSD
run.

| Operation | Payload bytes | Go ns/op | Zig ns/op | Result |
| --- | ---: | ---: | ---: | --- |
| Encode | 16 | 29.36 | 51.788 | Go 1.76x faster |
| Encode | 31 | 59.745 | 98.748 | Go 1.65x faster |
| Encode | 1024 | 2057 | 3382.612 | Go 1.64x faster |
| Encode | 65536 | 130505.5 | 217071.841 | Go 1.66x faster |
| Decode | 16 | 25.245 | 35.045 | Go 1.39x faster |
| Decode | 31 | 89.46 | 67.889 | Zig 1.32x faster |
| Decode | 1024 | 1414 | 2220.398 | Go 1.57x faster |
| Decode | 65536 | 89151 | 143926.964 | Go 1.61x faster |

### Intel Xeon w5-2455X, FreeBSD

Recorded 2026-09-12 with Go 1.26.7 and Zig 0.16.0, pinned to logical CPU 2.

| Operation | Payload bytes | Go ns/op | Zig ns/op | Result |
| --- | ---: | ---: | ---: | --- |
| Encode | 16 | 31.39 | 34.905 | Go 1.11x faster |
| Encode | 31 | 59.61 | 65.950 | Go 1.11x faster |
| Encode | 1024 | 1919 | 1603.091 | Zig 1.20x faster |
| Encode | 65536 | 131996 | 106704.178 | Zig 1.24x faster |
| Decode | 16 | 36.44 | 30.900 | Zig 1.18x faster |
| Decode | 31 | 103.8 | 58.677 | Zig 1.77x faster |
| Decode | 1024 | 1754 | 2341.094 | Go 1.33x faster |
| Decode | 65536 | 117002 | 120664.600 | Within 4% |

See [BENCHMARKS.md](BENCHMARKS.md) for complete methodology, allocating API
results, environment details, and interpretation.
