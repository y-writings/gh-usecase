package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"

	"github.com/y-writings/gh-usecase/internal/githubapi"
	"github.com/y-writings/gh-usecase/internal/prcount"
	"github.com/y-writings/gh-usecase/internal/prdetail"
	"github.com/y-writings/gh-usecase/internal/prlist"
	"github.com/y-writings/gh-usecase/internal/validation"
)

type graphQLClientFactory func() (githubapi.GraphQLClient, error)

func Run(argv []string, stdout io.Writer, stderr io.Writer) int {
	return runWithClientFactory(argv, stdout, stderr, githubapi.NewDefaultGraphQLClient)
}

func runWithClientFactory(argv []string, stdout io.Writer, stderr io.Writer, newClient graphQLClientFactory) int {
	parsed := ParseArgs(argv)

	if len(argv) == 0 || argv[0] == "--help" || argv[0] == "-h" {
		fmt.Fprintln(stdout, RootUsage)
		return 0
	}
	if len(parsed.Positionals) == 0 {
		fmt.Fprintln(stderr, RootUsage)
		return 1
	}

	command := parsed.Positionals[0]
	if !isKnownCommand(command) {
		fmt.Fprintln(stderr, RootUsage)
		fmt.Fprintf(stderr, "unknown command '%s'\n", command)
		return 1
	}

	switch command {
	case "pr-count":
		return runPrCount(parsed, stdout, stderr, newClient)
	case "pr-list":
		return runPrList(parsed, stdout, stderr, newClient)
	case "pr-detail":
		return runPrDetail(parsed, stdout, stderr, newClient)
	default:
		fmt.Fprintf(stderr, "command '%s' is not implemented yet\n", command)
		return 1
	}
}

func runPrCount(parsed ParsedArgs, stdout io.Writer, stderr io.Writer, newClient graphQLClientFactory) int {
	if parsed.Help {
		fmt.Fprintln(stdout, PrCountUsage)
		return 0
	}

	input := prcount.Input{
		Owner: parsed.Options["owner"],
		Name:  parsed.Options["name"],
	}
	if state, ok := parsed.Options["state"]; ok {
		input.State = &state
	}
	if err := prcount.Validate(input); err != nil {
		printCommandError(stderr, PrCountUsage, err)
		return 1
	}

	client, err := newClient()
	if err != nil {
		printExecutionError(stderr, err)
		return 1
	}

	output, err := prcount.Execute(context.Background(), client, input)
	if err != nil {
		printCommandError(stderr, PrCountUsage, err)
		return 1
	}

	encoded, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	fmt.Fprintln(stdout, string(encoded))
	return 0
}

func runPrList(parsed ParsedArgs, stdout io.Writer, stderr io.Writer, newClient graphQLClientFactory) int {
	if parsed.Help {
		fmt.Fprintln(stdout, PrListUsage)
		return 0
	}

	input := prlist.Input{
		Owner: parsed.Options["owner"],
		Name:  parsed.Options["name"],
	}
	if state, ok := parsed.Options["state"]; ok {
		input.State = &state
	}
	if after, ok := parsed.Options["after"]; ok {
		input.After = &after
	}
	if firstText, ok := parsed.Options["first"]; ok {
		first, err := strconv.Atoi(firstText)
		if err != nil {
			fmt.Fprintln(stderr, PrListUsage)
			fmt.Fprintln(stderr, "first must be an integer")
			return 1
		}
		input.First = &first
	}
	if err := prlist.Validate(input); err != nil {
		printCommandError(stderr, PrListUsage, err)
		return 1
	}

	client, err := newClient()
	if err != nil {
		printExecutionError(stderr, err)
		return 1
	}

	output, err := prlist.Execute(context.Background(), client, input)
	if err != nil {
		printCommandError(stderr, PrListUsage, err)
		return 1
	}

	encoded, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	fmt.Fprintln(stdout, string(encoded))
	return 0
}

func runPrDetail(parsed ParsedArgs, stdout io.Writer, stderr io.Writer, newClient graphQLClientFactory) int {
	if parsed.Help {
		fmt.Fprintln(stdout, PrDetailUsage)
		return 0
	}

	number, err := strconv.Atoi(parsed.Options["number"])
	if err != nil {
		fmt.Fprintln(stderr, PrDetailUsage)
		fmt.Fprintln(stderr, "number must be an integer")
		return 1
	}
	input := prdetail.Input{
		Owner:  parsed.Options["owner"],
		Name:   parsed.Options["name"],
		Number: number,
	}
	if filesFirstText, ok := parsed.Options["filesFirst"]; ok {
		filesFirst, err := strconv.Atoi(filesFirstText)
		if err != nil {
			fmt.Fprintln(stderr, PrDetailUsage)
			fmt.Fprintln(stderr, "filesFirst must be an integer")
			return 1
		}
		input.FilesFirst = &filesFirst
	}
	if err := prdetail.Validate(input); err != nil {
		printCommandError(stderr, PrDetailUsage, err)
		return 1
	}

	client, err := newClient()
	if err != nil {
		printExecutionError(stderr, err)
		return 1
	}

	output, err := prdetail.Execute(context.Background(), client, input)
	if err != nil {
		printCommandError(stderr, PrDetailUsage, err)
		return 1
	}

	encoded, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	fmt.Fprintln(stdout, string(encoded))
	return 0
}

func printCommandError(stderr io.Writer, usage string, err error) {
	var validationError validation.Error
	if errors.As(err, &validationError) {
		fmt.Fprintln(stderr, usage)
		fmt.Fprintln(stderr, validationError.Error())
		return
	}
	printExecutionError(stderr, err)
}

func printExecutionError(stderr io.Writer, err error) {
	fmt.Fprintf(stderr, "Failed to execute command: %v\n", err)
}

func isKnownCommand(command string) bool {
	switch command {
	case "pr-count", "pr-list", "pr-detail":
		return true
	default:
		return false
	}
}
