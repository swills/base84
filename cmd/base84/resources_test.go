package main

import (
	"bytes"
	"errors"
	"io"
	"io/fs"
	"os"
	"strings"
	"testing"
)

var (
	errTestClose  = errors.New("test close failure")
	errTestCreate = errors.New("test create failure")
	errTestStat   = errors.New("test stat failure")
	errTestWrite  = errors.New("test write failure")
)

type inputFileSystem struct {
	osFileSystem
	input fs.File
}

var _ fileSystem = inputFileSystem{}

func (filesystem inputFileSystem) open(string) (fs.File, error) {
	return filesystem.input, nil
}

type outputFileSystem struct {
	osFileSystem
	output io.WriteCloser
}

var _ fileSystem = outputFileSystem{}

func (filesystem outputFileSystem) create(string) (io.WriteCloser, error) {
	return filesystem.output, nil
}

type outputCreationFailureFileSystem struct {
	input fs.File
}

var _ fileSystem = outputCreationFailureFileSystem{}

func (filesystem outputCreationFailureFileSystem) open(string) (fs.File, error) {
	return filesystem.input, nil
}

func (outputCreationFailureFileSystem) stat(string) (fs.FileInfo, error) {
	return nil, os.ErrNotExist
}

func (outputCreationFailureFileSystem) create(string) (io.WriteCloser, error) {
	return nil, errTestCreate
}

type closeFailingInput struct {
	io.Reader
}

func (closeFailingInput) Close() error {
	return errTestClose
}

func (closeFailingInput) Stat() (fs.FileInfo, error) {
	return nil, errTestStat
}

type trackedInput struct {
	file   *os.File
	closed bool
}

func (input *trackedInput) Read(buffer []byte) (int, error) {
	return input.file.Read(buffer)
}

func (input *trackedInput) Close() error {
	input.closed = true

	return input.file.Close()
}

func (input *trackedInput) Stat() (fs.FileInfo, error) {
	return input.file.Stat()
}

type closeFailingOutput struct {
	writeErr error
	bytes.Buffer
}

func (output *closeFailingOutput) Write(value []byte) (int, error) {
	if output.writeErr != nil {
		return 0, output.writeErr
	}

	return output.Buffer.Write(value)
}

func (*closeFailingOutput) Close() error {
	return errTestClose
}

func TestCommandRunReportsInputCloseFailureWhenTransformSucceeds(t *testing.T) {
	filesystem := inputFileSystem{
		input: closeFailingInput{Reader: strings.NewReader("Hello, World!")},
	}

	var stdout, stderr bytes.Buffer

	application := command{
		stdin:  strings.NewReader("unused"),
		stdout: &stdout,
		stderr: &stderr,
		files:  filesystem,
	}

	code := application.run([]string{"input", "-"})

	if code != 1 {
		t.Fatalf("command.run(input close failure) returned %d, want 1", code)
	}

	if got := stdout.String(); got != "s@Etk'#Qedrxz+hhA\n" {
		t.Errorf("output before close failure = %q, want transformed output", got)
	}

	got := stderr.String()

	hasExpectedErrors := strings.Contains(got, "close input \"input\"") && strings.Contains(got, errTestClose.Error())
	if !hasExpectedErrors {
		t.Errorf("close failure stderr = %q, want path and close error", got)
	}
}

func TestCommandRunReportsOutputCloseFailureWhenTransformSucceeds(t *testing.T) {
	output := &closeFailingOutput{}
	filesystem := outputFileSystem{
		output: output,
	}

	var stdout, stderr bytes.Buffer

	application := command{
		stdin:  strings.NewReader("Hello, World!"),
		stdout: &stdout,
		stderr: &stderr,
		files:  filesystem,
	}

	code := application.run([]string{"-", "output"})

	if code != 1 {
		t.Fatalf("command.run(output close failure) returned %d, want 1", code)
	}

	if got := output.String(); got != "s@Etk'#Qedrxz+hhA\n" {
		t.Errorf("output before close failure = %q, want transformed output", got)
	}

	got := stderr.String()

	hasExpectedErrors := strings.Contains(got, "close output \"output\"") && strings.Contains(got, errTestClose.Error())
	if !hasExpectedErrors {
		t.Errorf("close failure stderr = %q, want path and close error", got)
	}
}

func TestCommandRunPreservesTransformErrorOverOutputCloseFailure(t *testing.T) {
	output := &closeFailingOutput{writeErr: errTestWrite}
	filesystem := outputFileSystem{
		output: output,
	}

	var stdout, stderr bytes.Buffer

	application := command{
		stdin:  strings.NewReader("Hello, World!"),
		stdout: &stdout,
		stderr: &stderr,
		files:  filesystem,
	}

	code := application.run([]string{"-", "output"})

	if code != 1 {
		t.Fatalf("command.run(write failure) returned %d, want 1", code)
	}

	got := stderr.String()

	hasExpectedError := strings.Contains(got, errTestWrite.Error()) && !strings.Contains(got, errTestClose.Error())
	if !hasExpectedError {
		t.Errorf("write failure stderr = %q, want write error without close error", got)
	}
}

func TestCommandRunClosesInputWhenOutputCreationFails(t *testing.T) {
	file, err := os.CreateTemp(t.TempDir(), "input")
	if err != nil {
		t.Fatalf("create input fixture: %v", err)
	}

	t.Cleanup(func() { _ = file.Close() })

	input := &trackedInput{file: file}
	filesystem := outputCreationFailureFileSystem{
		input: input,
	}

	var stdout, stderr bytes.Buffer

	application := command{
		stdin:  strings.NewReader("unused"),
		stdout: &stdout,
		stderr: &stderr,
		files:  filesystem,
	}

	code := application.run([]string{"input", "output"})

	if code != 1 {
		t.Fatalf("command.run(create failure) returned %d, want 1", code)
	}

	if !input.closed {
		t.Error("command.run(create failure) did not close its opened input")
	}

	got := stderr.String()

	hasExpectedErrors := strings.Contains(got, "create output \"output\"") && strings.Contains(got, errTestCreate.Error())
	if !hasExpectedErrors {
		t.Errorf("create failure stderr = %q, want path and creation error", got)
	}
}
