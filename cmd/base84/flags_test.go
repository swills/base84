package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestRunModeAliases(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
		args  []string
	}{
		{name: "short encode", args: []string{"-e"}, input: "Hello, World!", want: "s@Etk'#Qedrxz+hhA\n"},
		{name: "long encode", args: []string{"--encode"}, input: "Hello, World!", want: "s@Etk'#Qedrxz+hhA\n"},
		{name: "short decode", args: []string{"-d"}, input: "s@Etk'#Qedrxz+hhA\n", want: "Hello, World!"},
		{name: "long decode", args: []string{"--decode"}, input: "s@Etk'#Qedrxz+hhA\n", want: "Hello, World!"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer

			code := run(test.args, strings.NewReader(test.input), &stdout, &stderr)

			if code != 0 {
				t.Fatalf("run(%v) returned %d, want 0; stderr = %q", test.args, code, stderr.String())
			}

			if got := stdout.String(); got != test.want {
				t.Errorf("run(%v) output = %q, want %q", test.args, got, test.want)
			}

			if stderr.Len() != 0 {
				t.Errorf("run(%v) stderr = %q, want empty", test.args, stderr.String())
			}
		})
	}
}

func TestRunWrapAliases(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "short", args: []string{"-w", "4"}},
		{name: "attached short", args: []string{"-w4"}},
		{name: "long", args: []string{"--wrap", "4"}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer

			code := run(test.args, strings.NewReader("Hello, World!"), &stdout, &stderr)

			if code != 0 {
				t.Fatalf("run(%v) returned %d, want 0; stderr = %q", test.args, code, stderr.String())
			}

			if got, want := stdout.String(), "s@Et\nk'#Q\nedrx\nz+hh\nA\n"; got != want {
				t.Errorf("run(%v) output = %q, want %q", test.args, got, want)
			}
		})
	}
}

func TestRunIgnoreGarbageAliases(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "short", args: []string{"-d", "-i"}},
		{name: "long", args: []string{"--decode", "--ignore-garbage"}},
		{name: "single dash long", args: []string{"-decode", "-ignore-garbage"}},
		{name: "short cluster", args: []string{"-di"}},
		{name: "Fourmilab short", args: []string{"-d", "-n"}},
		{name: "Fourmilab short cluster", args: []string{"-dn"}},
		{name: "Fourmilab long", args: []string{"--decode", "--noerrcheck"}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer

			code := run(test.args, strings.NewReader("AY/y4A"), &stdout, &stderr)

			if code != 0 {
				t.Fatalf("run(%v) returned %d, want 0; stderr = %q", test.args, code, stderr.String())
			}

			if got, want := stdout.Bytes(), []byte{0x00, 0xe0, 0xff, 0x01}; !bytes.Equal(got, want) {
				t.Errorf("run(%v) output = %x, want %x", test.args, got, want)
			}

			if stderr.Len() != 0 {
				t.Errorf("run(%v) stderr = %q, want empty", test.args, stderr.String())
			}
		})
	}
}

func TestRunNoErrCheckPreservesStrictPaddingValidation(t *testing.T) {
	var stdout, stderr bytes.Buffer

	code := run([]string{"--decode", "--noerrcheck"}, strings.NewReader("/A?"), &stdout, &stderr)

	if code != 1 {
		t.Fatalf("run(--decode --noerrcheck invalid padding) returned %d, want 1", code)
	}

	if stdout.Len() != 0 {
		t.Errorf("invalid padding stdout = %x, want empty", stdout.Bytes())
	}

	if got := stderr.String(); !strings.Contains(got, "invalid padding") {
		t.Errorf("invalid padding stderr = %q, want padding error", got)
	}
}
