package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunFileOperands(t *testing.T) {
	tests := []struct {
		name       string
		operands   func(string, string) []string
		stdin      string
		wantStdout string
		wantFile   string
	}{
		{
			name:       "zero operands use standard streams",
			operands:   func(_, _ string) []string { return nil },
			stdin:      "Hello, World!",
			wantStdout: "s@Etk'#Qedrxz+hhA\n",
		},
		{
			name:       "one dash uses standard streams",
			operands:   func(_, _ string) []string { return []string{"-"} },
			stdin:      "Hello, World!",
			wantStdout: "s@Etk'#Qedrxz+hhA\n",
		},
		{
			name:       "one file reads file and writes stdout",
			operands:   func(input, _ string) []string { return []string{input} },
			stdin:      "ignored",
			wantStdout: "s@Etk'#Qedrxz+hhA\n",
		},
		{
			name:     "dash input writes output file",
			operands: func(_, output string) []string { return []string{"-", output} },
			stdin:    "Hello, World!",
			wantFile: "s@Etk'#Qedrxz+hhA\n",
		},
		{
			name:       "file input and dash output",
			operands:   func(input, _ string) []string { return []string{input, "-"} },
			stdin:      "ignored",
			wantStdout: "s@Etk'#Qedrxz+hhA\n",
		},
		{
			name:     "file input and file output",
			operands: func(input, output string) []string { return []string{input, output} },
			stdin:    "ignored",
			wantFile: "s@Etk'#Qedrxz+hhA\n",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			directory := t.TempDir()
			inputPath := filepath.Join(directory, "input")
			outputPath := filepath.Join(directory, "output")

			err := os.WriteFile(inputPath, []byte("Hello, World!"), 0o600)
			if err != nil {
				t.Fatalf("write input fixture: %v", err)
			}

			if test.wantFile != "" {
				err = os.WriteFile(outputPath, []byte("stale output that must be truncated"), 0o600)
				if err != nil {
					t.Fatalf("write output fixture: %v", err)
				}
			}

			var stdout, stderr bytes.Buffer

			code := run(test.operands(inputPath, outputPath), strings.NewReader(test.stdin), &stdout, &stderr)

			if code != 0 {
				t.Fatalf("run(file operands) returned %d, want 0; stderr = %q", code, stderr.String())
			}

			if got := stdout.String(); got != test.wantStdout {
				t.Errorf("stdout = %q, want %q", got, test.wantStdout)
			}

			if test.wantFile != "" {
				got, err := os.ReadFile(outputPath)
				if err != nil {
					t.Fatalf("read output: %v", err)
				}

				if string(got) != test.wantFile {
					t.Errorf("output file = %q, want %q", got, test.wantFile)
				}
			}
		})
	}
}

func TestRunRoundTripsThroughFiles(t *testing.T) {
	directory := t.TempDir()
	inputPath := filepath.Join(directory, "input")
	encodedPath := filepath.Join(directory, "encoded")
	decodedPath := filepath.Join(directory, "decoded")

	want := []byte{0x00, 0xe0, 0xff, 0x01}

	err := os.WriteFile(inputPath, want, 0o600)
	if err != nil {
		t.Fatalf("write input fixture: %v", err)
	}

	var stdout, stderr bytes.Buffer

	encodeCode := run([]string{inputPath, encodedPath}, strings.NewReader("ignored"), &stdout, &stderr)
	decodeCode := run([]string{"-d", encodedPath, decodedPath}, strings.NewReader("ignored"), &stdout, &stderr)

	if encodeCode != 0 || decodeCode != 0 {
		t.Fatalf("round trip exit codes = (%d, %d), want (0, 0); stderr = %q", encodeCode, decodeCode, stderr.String())
	}

	got, err := os.ReadFile(decodedPath)
	if err != nil {
		t.Fatalf("read decoded output: %v", err)
	}

	if !bytes.Equal(got, want) {
		t.Errorf("decoded output = %x, want %x", got, want)
	}
}

func TestRunReportsFileFailures(t *testing.T) {
	directory := t.TempDir()

	inputPath := filepath.Join(directory, "input")

	err := os.WriteFile(inputPath, []byte("input"), 0o600)
	if err != nil {
		t.Fatalf("write input fixture: %v", err)
	}

	tests := []struct {
		name        string
		wantContext string
		args        []string
	}{
		{
			name:        "nonexistent input",
			args:        []string{filepath.Join(directory, "missing")},
			wantContext: "open input",
		},
		{
			name:        "output creation failure",
			args:        []string{inputPath, filepath.Join(directory, "missing", "output")},
			wantContext: "create output",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer

			code := run(test.args, strings.NewReader("unused"), &stdout, &stderr)

			if code != 1 {
				t.Fatalf("run(%v) returned %d, want 1", test.args, code)
			}

			if stdout.Len() != 0 {
				t.Errorf("failure stdout = %q, want empty", stdout.String())
			}

			if got := stderr.String(); !strings.Contains(got, test.wantContext) {
				t.Errorf("failure stderr = %q, want %q context", got, test.wantContext)
			}
		})
	}
}

func TestRunRejectsSameInputAndOutputFile(t *testing.T) {
	directory := t.TempDir()

	inputPath := filepath.Join(directory, "input")

	err := os.WriteFile(inputPath, []byte("must remain intact"), 0o600)
	if err != nil {
		t.Fatalf("write input fixture: %v", err)
	}

	hardlinkPath := filepath.Join(directory, "hardlink")

	err = os.Link(inputPath, hardlinkPath)
	if err != nil {
		t.Fatalf("create hardlink fixture: %v", err)
	}

	symlinkPath := filepath.Join(directory, "symlink")

	err = os.Symlink(inputPath, symlinkPath)
	if err != nil {
		t.Fatalf("create symlink fixture: %v", err)
	}

	for _, outputPath := range []string{inputPath, hardlinkPath, symlinkPath} {
		t.Run(filepath.Base(outputPath), func(t *testing.T) {
			var stdout, stderr bytes.Buffer

			code := run([]string{inputPath, outputPath}, strings.NewReader("unused"), &stdout, &stderr)

			if code != 1 {
				t.Fatalf("run(same file) returned %d, want 1", code)
			}

			if got := stderr.String(); !strings.Contains(got, "same file") {
				t.Errorf("same-file stderr = %q, want same-file context", got)
			}

			got, err := os.ReadFile(inputPath)
			if err != nil {
				t.Fatalf("read preserved input: %v", err)
			}

			if string(got) != "must remain intact" {
				t.Errorf("input after rejection = %q, want original contents", got)
			}
		})
	}
}

func TestRunRejectsSameFileThroughSymlinkComponentAndParent(t *testing.T) {
	directory := t.TempDir()
	lexicalInputPath := filepath.Join(directory, "input")
	resolvedDirectory := filepath.Join(directory, "resolved")
	resolvedInputPath := filepath.Join(resolvedDirectory, "input")

	err := os.MkdirAll(filepath.Join(resolvedDirectory, "child"), 0o700)
	if err != nil {
		t.Fatalf("create resolved directory fixture: %v", err)
	}

	for _, path := range []string{lexicalInputPath, resolvedInputPath} {
		err = os.WriteFile(path, []byte("must remain intact"), 0o600)
		if err != nil {
			t.Fatalf("write input fixture %q: %v", path, err)
		}
	}

	linkPath := filepath.Join(directory, "link")

	err = os.Symlink(filepath.Join("resolved", "child"), linkPath)
	if err != nil {
		t.Fatalf("create symlink fixture: %v", err)
	}

	// Keep ".." after the symlink component so the kernel resolves it.
	operand := strings.Join([]string{linkPath, "..", "input"}, string(os.PathSeparator))

	var stdout, stderr bytes.Buffer

	code := run([]string{operand, operand}, strings.NewReader("unused"), &stdout, &stderr)

	if code != 1 {
		t.Errorf("run(symlink parent same file) returned %d, want 1", code)
	}

	if got := stderr.String(); !strings.Contains(got, "same file") {
		t.Errorf("same-file stderr = %q, want same-file context", got)
	}

	for _, path := range []string{lexicalInputPath, resolvedInputPath} {
		got, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read preserved input %q: %v", path, err)
		}

		if string(got) != "must remain intact" {
			t.Errorf("input %q after rejection = %q, want original contents", path, got)
		}
	}
}
