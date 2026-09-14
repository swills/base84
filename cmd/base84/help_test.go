package main

import (
	"bytes"
	"testing"
)

const expectedHelpOutput = `Usage: base84 [options] [input [output]]

Options:
  -e, --encode              encode input (default)
  -d, --decode              decode input
  -i, --ignore-garbage      ignore non-alphabet bytes when decoding
  -n, --noerrcheck          alias for --ignore-garbage
  -w, --wrap COLUMNS        wrap encoded output (0 disables wrapping)
  -h, -u, --help            show help
      --version             show version
`

func TestRunHelpPrintsExactOutputForEveryAlias(t *testing.T) {
	for _, option := range []string{"-h", "--help", "-u"} {
		t.Run(option, func(t *testing.T) {
			var stdout, stderr bytes.Buffer

			code := run([]string{option}, unreadableReader{}, &stdout, &stderr)

			if code != 0 {
				t.Fatalf("run(%s) returned %d, want 0", option, code)
			}

			if got := stdout.String(); got != expectedHelpOutput {
				t.Errorf("run(%s) stdout = %q, want %q", option, got, expectedHelpOutput)
			}

			if stderr.Len() != 0 {
				t.Errorf("run(%s) stderr = %q, want empty", option, stderr.String())
			}
		})
	}
}
