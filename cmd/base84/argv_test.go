package main

import (
	"bytes"
	"testing"

	"github.com/swills/base84/internal/cli"
)

func TestParseArgumentsParsesOptionsAfterOperands(t *testing.T) {
	tests := []struct {
		name          string
		inputPath     string
		outputPath    string
		args          []string
		mode          cli.Mode
		ignoreGarbage bool
	}{
		{
			name:       "after one operand",
			args:       []string{"input.base84", "-d"},
			inputPath:  "input.base84",
			outputPath: "-",
			mode:       cli.ModeDecode,
		},
		{
			name:          "after two operands",
			args:          []string{"input.base84", "output.bin", "-i", "-d"},
			inputPath:     "input.base84",
			outputPath:    "output.bin",
			mode:          cli.ModeDecode,
			ignoreGarbage: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer

			parsed := parseArguments(test.args, &stdout, &stderr)

			if !parsed.ready || parsed.exitCode != 0 {
				t.Fatalf("parseArguments(%v) = %+v; stderr = %q", test.args, parsed, stderr.String())
			}

			if got := parsed.invocation.inputPath; got != test.inputPath {
				t.Errorf("input path = %q, want %q", got, test.inputPath)
			}

			if got := parsed.invocation.outputPath; got != test.outputPath {
				t.Errorf("output path = %q, want %q", got, test.outputPath)
			}

			if got := parsed.invocation.options.Mode; got != test.mode {
				t.Errorf("mode = %v, want %v", got, test.mode)
			}

			if got := parsed.invocation.options.IgnoreGarbage; got != test.ignoreGarbage {
				t.Errorf("ignore garbage = %t, want %t", got, test.ignoreGarbage)
			}
		})
	}
}

func TestParseArgumentsPreservesDoubleDashOperands(t *testing.T) {
	tests := []struct {
		name          string
		inputPath     string
		outputPath    string
		args          []string
		mode          cli.Mode
		ignoreGarbage bool
	}{
		{
			name:       "options after delimiter are literal",
			args:       []string{"input.base84", "-d", "--", "-i"},
			inputPath:  "input.base84",
			outputPath: "-i",
			mode:       cli.ModeDecode,
		},
		{
			name:       "standard stream operand is preserved",
			args:       []string{"-d", "--", "-"},
			inputPath:  "-",
			outputPath: "-",
			mode:       cli.ModeDecode,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer

			parsed := parseArguments(test.args, &stdout, &stderr)

			if !parsed.ready || parsed.exitCode != 0 {
				t.Fatalf("parseArguments(%v) = %+v; stderr = %q", test.args, parsed, stderr.String())
			}

			if got := parsed.invocation.inputPath; got != test.inputPath {
				t.Errorf("input path = %q, want %q", got, test.inputPath)
			}

			if got := parsed.invocation.outputPath; got != test.outputPath {
				t.Errorf("output path = %q, want %q", got, test.outputPath)
			}

			if got := parsed.invocation.options.Mode; got != test.mode {
				t.Errorf("mode = %v, want %v", got, test.mode)
			}

			if got := parsed.invocation.options.IgnoreGarbage; got != test.ignoreGarbage {
				t.Errorf("ignore garbage = %t, want %t", got, test.ignoreGarbage)
			}
		})
	}
}

func TestParseArgumentsParsesWrapInShortCluster(t *testing.T) {
	tests := [][]string{
		{"-dw4"},
		{"-dw", "4"},
	}

	for _, args := range tests {
		var stdout, stderr bytes.Buffer

		parsed := parseArguments(args, &stdout, &stderr)

		if !parsed.ready || parsed.exitCode != 0 {
			t.Fatalf("parseArguments(%v) = %+v; stderr = %q", args, parsed, stderr.String())
		}

		if got, want := parsed.invocation.options.Mode, cli.ModeDecode; got != want {
			t.Errorf("mode = %v, want %v", got, want)
		}

		if got, want := parsed.invocation.options.WrapWidth, 4; got != want {
			t.Errorf("wrap width = %d, want %d", got, want)
		}
	}
}
