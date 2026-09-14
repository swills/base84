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

func (command command) transform(invocation invocation) (resultErr error) {
	input := command.stdin

	var openedInput fs.File

	if invocation.inputPath != "-" {
		file, err := command.files.open(invocation.inputPath)
		if err != nil {
			return fmt.Errorf("open input %q: %w", invocation.inputPath, err)
		}

		openedInput = file

		defer func() {
			closeErr := openedInput.Close()
			if closeErr != nil && resultErr == nil {
				resultErr = fmt.Errorf("close input %q: %w", invocation.inputPath, closeErr)
			}
		}()

		input = openedInput
	}

	output := command.stdout

	if invocation.outputPath != "-" {
		if openedInput != nil {
			err := command.rejectSameFile(openedInput, invocation)
			if err != nil {
				return err
			}
		}

		openedOutput, err := command.files.create(invocation.outputPath)
		if err != nil {
			return fmt.Errorf("create output %q: %w", invocation.outputPath, err)
		}

		defer func() {
			closeErr := openedOutput.Close()
			if closeErr != nil && resultErr == nil {
				resultErr = fmt.Errorf("close output %q: %w", invocation.outputPath, closeErr)
			}
		}()

		output = openedOutput
	}

	err := cli.Run(invocation.options, input, output)
	if err != nil {
		return fmt.Errorf("transform input: %w", err)
	}

	return nil
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
