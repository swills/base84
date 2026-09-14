// Zig 0.16: zig run -O ReleaseFast --dep base84 -Mroot=benchmark/zig_base84.zig -Mbase84=../zig-base84/src/root.zig

const std = @import("std");
const builtin = @import("builtin");
const base84 = @import("base84");

const codec = base84.standard;
const payload_sizes = [_]usize{ 16, 31, 1024, 65_536 };
const sample_count = 5;
const warmup_ns: u64 = 50_000_000;
const sample_target_ns: u64 = 500_000_000;
const max_iterations: u64 = 1 << 40;

comptime {
    if (builtin.mode != .ReleaseFast) {
        @compileError("build this benchmark with -O ReleaseFast");
    }
}

const Operation = enum {
    encode,
    decode,
};

const Buffers = struct {
    io: std.Io,
    payload: []u8,
    encoded_storage: []u8,
    encoded: []u8,
    decoded_storage: []u8,
};

pub fn main(init: std.process.Init) !void {
    std.debug.print("zig_version={s} mode=ReleaseFast samples={d}\n", .{
        builtin.zig_version_string,
        sample_count,
    });

    for (payload_sizes) |payload_len| {
        try runCase(init.gpa, init.io, payload_len);
    }
}

fn runCase(allocator: std.mem.Allocator, io: std.Io, payload_len: usize) !void {
    const payload = try allocator.alloc(u8, payload_len);
    defer allocator.free(payload);
    fillPayload(payload);

    const encoded_storage = try allocator.alloc(u8, codec.calcSizeUpperBound(payload_len));
    defer allocator.free(encoded_storage);
    const encoded = try codec.encode(encoded_storage, payload);

    const decoded_storage = try allocator.alloc(u8, codec.calcDecodedSizeUpperBound(encoded.len));
    defer allocator.free(decoded_storage);
    const decoded = try codec.decode(decoded_storage, encoded);
    if (!std.mem.eql(u8, payload, decoded)) return error.RoundTripMismatch;

    var buffers = Buffers{
        .io = io,
        .payload = payload,
        .encoded_storage = encoded_storage,
        .encoded = encoded,
        .decoded_storage = decoded_storage,
    };
    try benchmark(.encode, &buffers);
    try benchmark(.decode, &buffers);
}

fn fillPayload(payload: []u8) void {
    for (payload, 0..) |*byte, index| {
        byte.* = @truncate((index *% 131 +% 17) & 0xff);
    }
}

fn benchmark(operation: Operation, buffers: *Buffers) !void {
    try warmUp(operation, buffers);
    const iterations = try calibrate(operation, buffers);

    var samples: [sample_count]u64 = undefined;
    for (&samples) |*elapsed| {
        elapsed.* = try measure(operation, buffers, iterations);
    }

    const median_elapsed = median(samples);
    if (median_elapsed == 0) return error.TimerResolutionTooLow;
    const ns_per_op = @as(f64, @floatFromInt(median_elapsed)) /
        @as(f64, @floatFromInt(iterations));
    const mb_per_second = @as(f64, @floatFromInt(buffers.payload.len)) * 1_000.0 / ns_per_op;
    const output = switch (operation) {
        .encode => buffers.encoded,
        .decode => buffers.decoded_storage[0..buffers.payload.len],
    };

    std.debug.print(
        "operation={s} payload_bytes={d} median_ns_per_op={d:.3} payload_mb_per_s={d:.3} iterations={d} check={x}\n",
        .{ @tagName(operation), buffers.payload.len, ns_per_op, mb_per_second, iterations, checksum(output) },
    );
}

fn warmUp(operation: Operation, buffers: *Buffers) !void {
    const start = benchTime(buffers.io);
    while (benchTime(buffers.io) - start < warmup_ns) {
        _ = try invoke(operation, buffers);
    }
}

fn calibrate(operation: Operation, buffers: *Buffers) !u64 {
    var iterations: u64 = 1;
    while (true) {
        const elapsed = try measure(operation, buffers, iterations);
        if (elapsed >= sample_target_ns or iterations == max_iterations) return iterations;

        const scale = @max(@as(u64, 2), @min(@as(u64, 16), sample_target_ns / @max(elapsed, 1)));
        iterations = @min(max_iterations, iterations *| scale);
    }
}

fn measure(operation: Operation, buffers: *Buffers, iterations: u64) !u64 {
    const start = benchTime(buffers.io);
    var index: u64 = 0;
    while (index < iterations) : (index += 1) {
        _ = try invoke(operation, buffers);
    }
    return @intCast(benchTime(buffers.io) - start);
}

fn benchTime(io: std.Io) i96 {
    return std.Io.Clock.awake.now(io).nanoseconds;
}

fn invoke(operation: Operation, buffers: *Buffers) !usize {
    const result = switch (operation) {
        .encode => try codec.encode(buffers.encoded_storage, buffers.payload),
        .decode => try codec.decode(buffers.decoded_storage, buffers.encoded),
    };
    std.mem.doNotOptimizeAway(result);
    return result.len;
}

fn median(samples: [sample_count]u64) u64 {
    var sorted = samples;
    for (1..sorted.len) |index| {
        var position = index;
        while (position > 0 and sorted[position] < sorted[position - 1]) : (position -= 1) {
            std.mem.swap(u64, &sorted[position], &sorted[position - 1]);
        }
    }
    return sorted[sample_count / 2];
}

fn checksum(bytes: []const u8) u64 {
    var hash: u64 = 14_695_981_039_346_656_037;
    for (bytes) |byte| {
        hash = (hash ^ byte) *% 1_099_511_628_211;
    }
    return hash;
}
