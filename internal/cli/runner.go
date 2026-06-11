package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strconv"

	"github.com/y-writings/gh-usecase/internal/codeqldefaultsetup"
	"github.com/y-writings/gh-usecase/internal/githubapi"
	"github.com/y-writings/gh-usecase/internal/prcount"
	"github.com/y-writings/gh-usecase/internal/prdetail"
	"github.com/y-writings/gh-usecase/internal/prlist"
	"github.com/y-writings/gh-usecase/internal/pullrequestcreationpolicy"
	"github.com/y-writings/gh-usecase/internal/validation"
)

type graphQLClientFactory func() (githubapi.GraphQLClient, error)
type restClientFactory func() (githubapi.RESTClient, error)

func Run(argv []string, stdout io.Writer, stderr io.Writer) int {
	return runWithClientFactories(argv, stdout, stderr, githubapi.NewDefaultGraphQLClient, githubapi.NewDefaultRESTClient)
}

func runWithClientFactory(argv []string, stdout io.Writer, stderr io.Writer, newClient graphQLClientFactory) int {
	return runWithClientFactories(argv, stdout, stderr, newClient, githubapi.NewDefaultRESTClient)
}

func runWithClientFactories(argv []string, stdout io.Writer, stderr io.Writer, newGraphQLClient graphQLClientFactory, newRESTClient restClientFactory) int {
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
		return runPrCount(parsed, stdout, stderr, newGraphQLClient)
	case "pr-list":
		return runPrList(parsed, stdout, stderr, newGraphQLClient)
	case "pr-detail":
		return runPrDetail(parsed, stdout, stderr, newGraphQLClient)
	case "codeql-default-setup":
		return runCodeQLDefaultSetup(parsed, stdout, stderr, newRESTClient)
	case "pull-request-creation-policy":
		return runPullRequestCreationPolicy(parsed, stdout, stderr, newRESTClient)
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

func runCodeQLDefaultSetup(parsed ParsedArgs, stdout io.Writer, stderr io.Writer, newClient restClientFactory) int {
	if parsed.Help {
		fmt.Fprintln(stdout, CodeQLDefaultSetupUsage)
		return 0
	}

	if err := rejectUnsupportedOptions(parsed, []string{"owner", "repo", "languages"}); err != nil {
		printCommandError(stderr, CodeQLDefaultSetupUsage, err)
		return 1
	}
	if parsed.OptionOccurrences["languages"] > 1 {
		printCommandError(stderr, CodeQLDefaultSetupUsage, validation.New("languages may be specified only once"))
		return 1
	}

	input := codeqldefaultsetup.Input{
		Owner:     parsed.Options["owner"],
		Repo:      parsed.Options["repo"],
		Languages: parsed.Options["languages"],
	}
	if _, err := codeqldefaultsetup.Validate(input); err != nil {
		printCommandError(stderr, CodeQLDefaultSetupUsage, err)
		return 1
	}

	client, err := newClient()
	if err != nil {
		printExecutionError(stderr, err)
		return 1
	}

	output, err := codeqldefaultsetup.Execute(context.Background(), client, input)
	if err != nil {
		printCommandError(stderr, CodeQLDefaultSetupUsage, err)
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

func runPullRequestCreationPolicy(parsed ParsedArgs, stdout io.Writer, stderr io.Writer, newClient restClientFactory) int {
	if parsed.Help {
		fmt.Fprintln(stdout, PullRequestCreationPolicyUsage)
		return 0
	}

	if err := rejectUnsupportedOptions(parsed, []string{"owner", "repo", "policy"}); err != nil {
		printCommandError(stderr, PullRequestCreationPolicyUsage, err)
		return 1
	}
	for _, option := range []string{"owner", "repo", "policy"} {
		if parsed.OptionOccurrences[option] > 1 {
			printCommandError(stderr, PullRequestCreationPolicyUsage, validation.New(option+" may be specified only once"))
			return 1
		}
	}

	input := pullrequestcreationpolicy.Input{
		Owner:  parsed.Options["owner"],
		Repo:   parsed.Options["repo"],
		Policy: parsed.Options["policy"],
	}
	if _, err := pullrequestcreationpolicy.Validate(input); err != nil {
		printCommandError(stderr, PullRequestCreationPolicyUsage, err)
		return 1
	}

	client, err := newClient()
	if err != nil {
		printExecutionError(stderr, err)
		return 1
	}

	output, err := pullrequestcreationpolicy.Execute(context.Background(), client, input)
	if err != nil {
		printCommandError(stderr, PullRequestCreationPolicyUsage, err)
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

func rejectUnsupportedOptions(parsed ParsedArgs, allowed []string) error {
	allowedOptions := make(map[string]struct{}, len(allowed))
	for _, option := range allowed {
		allowedOptions[option] = struct{}{}
	}

	unknown := make([]string, 0)
	for option := range parsed.OptionOccurrences {
		if _, ok := allowedOptions[option]; !ok {
			unknown = append(unknown, option)
		}
	}
	if len(unknown) == 0 {
		return nil
	}
	sort.Strings(unknown)
	return validation.New("unknown option --" + unknown[0])
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
	case "pr-count", "pr-list", "pr-detail", "codeql-default-setup", "pull-request-creation-policy":
		return true
	default:
		return false
	}
}
