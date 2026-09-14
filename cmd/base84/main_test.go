package main

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"
)

type unreadableReader struct{}

func (unreadableReader) Read([]byte) (int, error) {
	return 0, errors.New("stdin was read")
}

type failingVersionWriter struct{}

func (failingVersionWriter) Write([]byte) (int, error) {
	return 0, errors.New("stdout write failed")
}

type shortVersionWriter struct{}

func (shortVersionWriter) Write(buffer []byte) (int, error) {
	return len(buffer) - 1, nil
}

func TestWriteVersionReportsShortWrite(t *testing.T) {
	err := writeVersion(shortVersionWriter{})

	if !errors.Is(err, io.ErrShortWrite) {
		t.Fatalf("writeVersion(short writer) error = %v, want %v", err, io.ErrShortWrite)
	}
}

func TestRunVersionUsesDefaultVersionWithoutReadingStdin(t *testing.T) {
	var stdout, stderr bytes.Buffer

	code := run([]string{"-version"}, unreadableReader{}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("run(-version) returned %d, want 0; stderr = %q", code, stderr.String())
	}

	if got, want := stdout.String(), "base84 devel\n"; got != want {
		t.Errorf("version output = %q, want %q", got, want)
	}

	if stderr.Len() != 0 {
		t.Errorf("version stderr = %q, want empty", stderr.String())
	}
}

func TestRunVersionUsesInjectedVersion(t *testing.T) {
	originalVersion := version
	version = "1.2.3"

	t.Cleanup(func() { version = originalVersion })

	var stdout, stderr bytes.Buffer

	code := run([]string{"--version"}, strings.NewReader("must not be read"), &stdout, &stderr)

	if code != 0 {
		t.Fatalf("run(--version) returned %d, want 0", code)
	}

	if got, want := stdout.String(), "base84 1.2.3\n"; got != want {
		t.Errorf("version output = %q, want %q", got, want)
	}
}

func TestRunVersionReportsOutputFailure(t *testing.T) {
	var stderr bytes.Buffer

	code := run([]string{"-version"}, strings.NewReader("unused"), failingVersionWriter{}, &stderr)

	if code != 1 {
		t.Fatalf("run(-version) returned %d, want 1", code)
	}

	if got := stderr.String(); !strings.Contains(got, "stdout write failed") {
		t.Errorf("version error = %q, want stdout failure", got)
	}
}

func TestRunEncode(t *testing.T) {
	var stdout, stderr bytes.Buffer

	code := run(nil, strings.NewReader("Hello, World!"), &stdout, &stderr)

	if code != 0 {
		t.Fatalf("run(encode) returned %d, want 0; stderr = %q", code, stderr.String())
	}

	if got, want := stdout.String(), "s@Etk'#Qedrxz+hhA\n"; got != want {
		t.Errorf("encode output = %q, want %q", got, want)
	}
}

func TestRunDecode(t *testing.T) {
	var stdout, stderr bytes.Buffer

	code := run([]string{"-d"}, strings.NewReader("AYy4A"), &stdout, &stderr)

	if code != 0 {
		t.Fatalf("run(-d) returned %d, want 0; stderr = %q", code, stderr.String())
	}

	if got, want := stdout.Bytes(), []byte{0x00, 0xe0, 0xff, 0x01}; !bytes.Equal(got, want) {
		t.Errorf("decode output = %x, want %x", got, want)
	}
}

func TestRunTransformError(t *testing.T) {
	var stdout, stderr bytes.Buffer

	code := run([]string{"-d"}, strings.NewReader("AA/"), &stdout, &stderr)

	if code != 1 {
		t.Fatalf("run(-d invalid) returned %d, want 1", code)
	}

	if !strings.Contains(stderr.String(), "decode input") {
		t.Errorf("transform error = %q, want decode context", stderr.String())
	}
}

func TestRunFlagParseError(t *testing.T) {
	var stdout, stderr bytes.Buffer

	code := run([]string{"-unknown"}, strings.NewReader("unused"), &stdout, &stderr)

	if code != 2 {
		t.Fatalf("run(-unknown) returned %d, want 2", code)
	}

	if stderr.Len() == 0 {
		t.Error("flag parse error wrote no stderr output")
	}
}

func TestRunUsageErrorsDoNotReadInput(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "mixed short aliases", args: []string{"-e", "-d"}},
		{name: "mixed long aliases", args: []string{"--encode", "--decode"}},
		{name: "short encode long decode", args: []string{"-e", "--decode"}},
		{name: "long encode short decode", args: []string{"--encode", "-d"}},
		{name: "negative short wrap", args: []string{"-w", "-1"}},
		{name: "negative long wrap", args: []string{"--wrap", "-2"}},
		{name: "negative attached wrap", args: []string{"-w-1"}},
		{name: "missing wrap", args: []string{"-w"}},
		{name: "missing clustered wrap", args: []string{"-iw"}},
		{name: "malformed attached wrap", args: []string{"-wnope"}},
		{name: "conflicting cluster", args: []string{"-ed"}},
		{name: "unknown option after operand", args: []string{"input", "-unknown"}},
		{name: "too many operands", args: []string{"one", "two", "three"}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer

			code := run(test.args, unreadableReader{}, &stdout, &stderr)

			if code != 2 {
				t.Fatalf("run(%v) returned %d, want 2", test.args, code)
			}

			if stdout.Len() != 0 {
				t.Errorf("usage error stdout = %q, want empty", stdout.String())
			}

			if got := stderr.String(); !strings.Contains(got, "Usage: base84 [options] [input [output]]") {
				t.Errorf("usage error stderr = %q, want usage line", got)
			}
		})
	}
}

func TestRunVersionShortCircuitsOperandHandling(t *testing.T) {
	var stdout, stderr bytes.Buffer

	code := run(
		[]string{"--version", "does-not-exist", "also-does-not-exist", "extra"},
		unreadableReader{},
		&stdout,
		&stderr,
	)

	if code != 0 {
		t.Fatalf("run(--version operands) returned %d, want 0; stderr = %q", code, stderr.String())
	}

	if got, want := stdout.String(), "base84 devel\n"; got != want {
		t.Errorf("version output = %q, want %q", got, want)
	}

	if stderr.Len() != 0 {
		t.Errorf("version stderr = %q, want empty", stderr.String())
	}
}
