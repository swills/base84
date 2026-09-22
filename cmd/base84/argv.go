package main

import "strings"

type normalizedOption struct {
	options   []string
	nextIndex int
}

func normalizeArguments(args []string) []string {
	options := make([]string, 0, len(args))
	operands := make([]string, 0, len(args))

	for index := 0; index < len(args); index++ {
		argument := args[index]
		if argument == "--" {
			operands = append(operands, args[index+1:]...)

			break
		}

		if argument == "-" || !strings.HasPrefix(argument, "-") {
			operands = append(operands, argument)

			continue
		}

		normalized := appendNormalizedOption(options, args, index)
		options, index = normalized.options, normalized.nextIndex
	}

	normalized := make([]string, 0, len(options)+len(operands)+1)
	normalized = append(normalized, options...)
	normalized = append(normalized, "--")
	normalized = append(normalized, operands...)

	return normalized
}

func appendNormalizedOption(options, args []string, index int) normalizedOption {
	argument := args[index]
	if isWrapOption(argument) {
		options = append(options, argument)

		if index+1 < len(args) {
			index++
			options = append(options, args[index])
		}

		return normalizedOption{options: options, nextIndex: index}
	}

	if isPreservedOption(argument) || strings.HasPrefix(argument, "--") {
		return normalizedOption{options: append(options, argument), nextIndex: index}
	}

	expanded, ok := expandShortOptions(argument)
	if !ok {
		return normalizedOption{options: append(options, argument), nextIndex: index}
	}

	options = append(options, expanded...)
	if expanded[len(expanded)-1] == "-w" && index+1 < len(args) {
		index++
		options = append(options, args[index])
	}

	return normalizedOption{options: options, nextIndex: index}
}

func isWrapOption(argument string) bool {
	return argument == "-w" || argument == "-wrap" || argument == "--wrap"
}

func isPreservedOption(argument string) bool {
	for _, name := range []string{
		"-decode", "-encode", "-ignore-garbage", "-noerrcheck", "-version", "-wrap",
		"-d", "-e", "-h", "-i", "-n", "-u", "-w",
	} {
		if argument == name || strings.HasPrefix(argument, name+"=") {
			return true
		}
	}

	return false
}

func expandShortOptions(argument string) ([]string, bool) {
	body := argument[1:]
	expanded := make([]string, 0, len(body))

	for index := 0; index < len(body); index++ {
		option := body[index]
		switch option {
		case 'd', 'e', 'h', 'i', 'n', 'u':
			expanded = append(expanded, "-"+string(option))
		case 'w':
			expanded = append(expanded, "-w")
			if index+1 < len(body) {
				expanded = append(expanded, body[index+1:])
			}

			return expanded, true
		default:
			return nil, false
		}
	}

	return expanded, true
}
