package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/swills/base84/internal/cli"
)

var version = "devel"

const helpOutput = `Usage: base84 [options] [input [output]]

Options:
  -e, --encode              encode input (default)
  -d, --decode              decode input
  -i, --ignore-garbage      ignore non-alphabet bytes when decoding
  -n, --noerrcheck          alias for --ignore-garbage
  -w, --wrap COLUMNS        wrap encoded output (0 disables wrapping)
  -h, -u, --help            show help
      --version             show version
`

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	application := command{
		stdin:  stdin,
		stdout: stdout,
		stderr: stderr,
		files:  osFileSystem{},
	}

	return application.run(args)
}

type invocation struct {
	inputPath  string
	outputPath string
	options    cli.Options
	version    bool
}

type parseOutcome struct {
	invocation invocation
	exitCode   int
	ready      bool
}

type argumentValues struct {
	encode        bool
	decode        bool
	help          bool
	ignoreGarbage bool
	showVersion   bool
	wrapWidth     int
}

type command struct {
	stdin  io.Reader
	stdout io.Writer
	stderr io.Writer
	files  fileSystem
}

func (command command) run(args []string) int {
	parsed := parseArguments(args, command.stdout, command.stderr)
	if !parsed.ready {
		return parsed.exitCode
	}

	if parsed.invocation.version {
		err := writeVersion(command.stdout)
		if err != nil {
			_, _ = fmt.Fprintln(command.stderr, err)

			return 1
		}

		return 0
	}

	err := command.transform(parsed.invocation)
	if err != nil {
		_, _ = fmt.Fprintln(command.stderr, err)

		return 1
	}

	return 0
}

func parseArguments(args []string, stdout, stderr io.Writer) parseOutcome {
	var (
		values          argumentValues
		emptyInvocation invocation
	)

	flags := newArgumentFlagSet(&values, stderr)

	err := flags.Parse(normalizeArguments(args))
	if err != nil {
		return parseOutcome{invocation: emptyInvocation, exitCode: 2, ready: false}
	}

	if values.help {
		flags.SetOutput(stdout)
		flags.Usage()

		return parseOutcome{invocation: emptyInvocation, exitCode: 0, ready: false}
	}

	if values.showVersion {
		versionInvocation := emptyInvocation
		versionInvocation.version = true

		return parseOutcome{invocation: versionInvocation, exitCode: 0, ready: true}
	}

	if values.encode && values.decode {
		reportUsage(flags, "encode and decode modes cannot be used together")

		return parseOutcome{invocation: emptyInvocation, exitCode: 2, ready: false}
	}

	if values.wrapWidth < 0 {
		reportUsage(flags, "wrap width must be nonnegative")

		return parseOutcome{invocation: emptyInvocation, exitCode: 2, ready: false}
	}

	operands := flags.Args()
	if len(operands) > 2 {
		reportUsage(flags, "expected at most two operands")

		return parseOutcome{invocation: emptyInvocation, exitCode: 2, ready: false}
	}

	parsed := invocation{
		options: cli.Options{
			Mode:          cli.ModeEncode,
			IgnoreGarbage: values.ignoreGarbage,
			WrapWidth:     values.wrapWidth,
		},
		inputPath:  "-",
		outputPath: "-",
		version:    false,
	}
	if values.decode {
		parsed.options.Mode = cli.ModeDecode
	}

	if len(operands) > 0 {
		parsed.inputPath = operands[0]
	}

	if len(operands) > 1 {
		parsed.outputPath = operands[1]
	}

	return parseOutcome{invocation: parsed, exitCode: 0, ready: true}
}

func newArgumentFlagSet(values *argumentValues, output io.Writer) *flag.FlagSet {
	flags := flag.NewFlagSet("base84", flag.ContinueOnError)
	flags.SetOutput(output)
	flags.Usage = func() {
		_, _ = io.WriteString(flags.Output(), helpOutput)
	}

	flags.BoolVar(&values.encode, "e", false, "encode input (default)")
	flags.BoolVar(&values.encode, "encode", false, "encode input (default)")
	flags.BoolVar(&values.decode, "d", false, "decode input")
	flags.BoolVar(&values.decode, "decode", false, "decode input")
	flags.BoolVar(&values.help, "h", false, "show help")
	flags.BoolVar(&values.help, "help", false, "show help")
	flags.BoolVar(&values.help, "u", false, "show help")
	flags.BoolVar(&values.ignoreGarbage, "i", false, "ignore non-alphabet bytes when decoding")
	flags.BoolVar(&values.ignoreGarbage, "ignore-garbage", false, "ignore non-alphabet bytes when decoding")
	flags.BoolVar(&values.ignoreGarbage, "n", false, "ignore non-alphabet bytes when decoding")
	flags.BoolVar(&values.ignoreGarbage, "noerrcheck", false, "ignore non-alphabet bytes when decoding")
	flags.IntVar(&values.wrapWidth, "w", 0, "wrap encoded output at `columns` (0 disables wrapping)")
	flags.IntVar(&values.wrapWidth, "wrap", 0, "wrap encoded output at `columns` (0 disables wrapping)")
	flags.BoolVar(&values.showVersion, "version", false, "show version")

	return flags
}

func reportUsage(flags *flag.FlagSet, message string) {
	_, _ = fmt.Fprintf(flags.Output(), "base84: %s\n", message)
	flags.Usage()
}

func writeVersion(output io.Writer) error {
	versionLine := fmt.Sprintf("base84 %s\n", version)

	written, err := io.WriteString(output, versionLine)
	if err != nil {
		return fmt.Errorf("write version: %w", err)
	}

	if written != len(versionLine) {
		return io.ErrShortWrite
	}

	return nil
}
