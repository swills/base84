package main

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"

	"github.com/swills/base84/internal/cli"
)

var errSameFile = errors.New("input and output refer to the same file")

type fileSystem interface {
	open(path string) (fs.File, error)
	create(path string) (io.WriteCloser, error)
	stat(path string) (fs.FileInfo, error)
}

type osFileSystem struct{}

func (osFileSystem) open(path string) (fs.File, error) {
	//nolint:gosec,wrapcheck // Direct stdlib adapter preserves the os.Open error for caller context.
	return os.Open(path)
}

func (osFileSystem) create(path string) (io.WriteCloser, error) {
	//nolint:gosec,wrapcheck // Direct stdlib adapter preserves the os.Create error for caller context.
	return os.Create(path)
}

func (osFileSystem) stat(path string) (fs.FileInfo, error) {
	//nolint:wrapcheck // Direct stdlib adapter preserves the os.Stat error for caller context.
	return os.Stat(path)
}

func (command command) transform(invocation invocation) error {
	input, openedInput, err := command.openInput(invocation.inputPath)
	if err != nil {
		return err
	}

	output, openedOutput, resultErr := command.openOutput(openedInput, invocation)
	if resultErr != nil {
		if openedInput != nil {
			resultErr = closeTransformFile(openedInput, "input", invocation.inputPath, resultErr)
		}

		return resultErr
	}

	resultErr = cli.Run(invocation.options, input, output)
	if resultErr != nil {
		resultErr = fmt.Errorf("transform input: %w", resultErr)
	}

	if openedOutput != nil {
		resultErr = closeTransformFile(openedOutput, "output", invocation.outputPath, resultErr)
	}

	if openedInput != nil {
		resultErr = closeTransformFile(openedInput, "input", invocation.inputPath, resultErr)
	}

	return resultErr
}

func (command command) openInput(path string) (io.Reader, fs.File, error) {
	if path == "-" {
		return command.stdin, nil, nil
	}

	input, err := command.files.open(path)
	if err != nil {
		return nil, nil, fmt.Errorf("open input %q: %w", path, err)
	}

	return input, input, nil
}

func (command command) openOutput(input fs.File, invocation invocation) (io.Writer, io.WriteCloser, error) {
	if invocation.outputPath == "-" {
		return command.stdout, nil, nil
	}

	if input != nil {
		err := command.rejectSameFile(input, invocation)
		if err != nil {
			return nil, nil, err
		}
	}

	output, err := command.files.create(invocation.outputPath)
	if err != nil {
		return nil, nil, fmt.Errorf("create output %q: %w", invocation.outputPath, err)
	}

	return output, output, nil
}

func closeTransformFile(closer io.Closer, kind, path string, resultErr error) error {
	closeErr := closer.Close()
	if closeErr != nil && resultErr == nil {
		return fmt.Errorf("close %s %q: %w", kind, path, closeErr)
	}

	return resultErr
}

func (command command) rejectSameFile(input fs.File, invocation invocation) error {
	inputInfo, err := input.Stat()
	if err != nil {
		return fmt.Errorf("inspect input %q: %w", invocation.inputPath, err)
	}

	outputInfo, err := command.files.stat(invocation.outputPath)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}

	if err != nil {
		return fmt.Errorf("inspect output %q: %w", invocation.outputPath, err)
	}

	if os.SameFile(inputInfo, outputInfo) {
		return fmt.Errorf(
			"input %q and output %q: %w",
			invocation.inputPath,
			invocation.outputPath,
			errSameFile,
		)
	}

	return nil
}
