# Benchmarks

Recorded 2026-09-12 on FreeBSD 16.0-CURRENT amd64, Intel Xeon w5-2455X,
logical CPU 2. Go was 1.26.7 with `GOMAXPROCS=1`; Zig was 0.16.0 in
`ReleaseFast` mode. `powerd` was stopped and `dev.hwpstate_intel.2.epp` was
set to `0` during the recorded runs.

## Reproduce

Run the suites sequentially on the same otherwise-idle host. Pin them to one
logical CPU when `cpuset` is available:

```sh
cpuset -l 2 task bench
cpuset -l 2 task bench-zig
```

The pinned commands are:

```sh
GOMAXPROCS=1 go test -run '^$' -bench '^Benchmark(Encode|Decode)(Allocate|Reuse)$' -benchmem -benchtime=500ms -count=5 .
zig run -O ReleaseFast --dep base84 -Mroot=benchmark/zig_base84.zig -Mbase84="$ZIG_BASE84_ROOT/src/root.zig" --cache-dir .zig-cache
```

`task bench-zig` defaults `ZIG_BASE84_ROOT` to `../zig-base84`. Override it
when needed, for example `ZIG_BASE84_ROOT=/path/to/zig-base84 task bench-zig`.

## Method

Each case uses bytes `i` defined by `(i*131+17)&0xff` and payload sizes 16,
31, 1024, and 65536 bytes. Setup and buffer allocation occur before timing.
Each suite takes five 500ms samples and reports the median. Throughput uses
decoded payload bytes. Go `MB/s` is decimal payload MB/s, and the Zig harness
emits the matching `payload_mb_per_s` field.

The primary comparison is Go `Encoding.Encode` and `Encoding.Decode` with
fixed destination buffers against Zig fixed buffers. Go allocating convenience
APIs are reported separately. The Zig harness does not measure allocating APIs.

The final table uses medians from Go decode and encode suites run immediately
before and after the Zig suite, including samples affected by observed CPU
frequency changes. Isolated A/B runs were used to decide which implementation
changes to retain: the 256-byte decode lookup improved bulk decode by about
13–14%; grouped encode output improved bulk encode by about 13%; narrowing digit
arithmetic to `uint32` improved it by a further 8%. A looped grouped decode
writer was 4–5% slower and was rejected, but inlineable three- and four-byte
writers lowered best-frequency decode floors by about 5–7%. In one adjacent
candidate-then-baseline comparison using longer two-second samples, their
medians improved by about 5% at 1024 bytes and 13% at 65536 bytes. Specialized
five-digit encode emission had an 11–12% lower best-frequency floor than the
generic `uint32` loop. Transitions between CPU frequency states made individual
sample groups noisy.

For full decode groups, moving invalid-character formatting out of `readChunk`
did not make it inlineable and produced no consistent gain. A fused
five-character reader with one combined validity check, fixed-place `uint32`
arithmetic, and one dominating bounds check improved stable adjacent medians by
about 24–25% at 1024 and 65536 bytes. Horner arithmetic was effectively tied at
65536 bytes but about 17% slower at 1024 bytes, so the fixed-place form was
retained. Invalid groups and short tails still use the generic reader.

## Fixed-Buffer Medians

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

## Allocating Go API Medians

| Operation | Payload bytes | ns/op | B/op | allocs/op |
| --- | ---: | ---: | ---: | ---: |
| Encode | 16 | 67.60 | 48 | 2 |
| Encode | 31 | 100.5 | 96 | 2 |
| Encode | 1024 | 2148 | 2816 | 2 |
| Encode | 65536 | 131468 | 180224 | 2 |
| Decode | 16 | 42.65 | 16 | 1 |
| Decode | 31 | 132.2 | 32 | 1 |
| Decode | 1024 | 1631 | 1024 | 1 |
| Decode | 65536 | 105162 | 65536 | 1 |

## Interpretation and Limits

On this sequential host run, Zig's fixed-buffer implementation leads bulk
encode, while Go leads 1024-byte decode and 65536-byte decode is within 4%.
Go leads 16- and 31-byte encode, while Zig leads decode at those sizes. Go reuse
is 0 allocs/op at every listed size; the allocating convenience APIs have the
costs shown above.

These results are illustrative, not a universal language or library ranking.
Compiler versions, CPU model, frequency behavior, CPU affinity, and background
load can change them. Keep the host conditions and CPU pinning comparable when
making a new run. In repeated runs during development, the 16-byte ranking
flipped and larger Go results also moved materially despite CPU pinning. Treat
small differences as noise unless an interleaved experiment reproduces them.
