package codeqldefaultsetup

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"

	"github.com/y-writings/gh-usecase/internal/githubapi"
	"github.com/y-writings/gh-usecase/internal/validation"
)

var allowedLanguages = map[string]struct{}{
	"actions":               {},
	"c-cpp":                 {},
	"csharp":                {},
	"go":                    {},
	"java-kotlin":           {},
	"javascript-typescript": {},
	"python":                {},
	"ruby":                  {},
	"swift":                 {},
}

var allowedLanguageList = []string{
	"actions",
	"c-cpp",
	"csharp",
	"go",
	"java-kotlin",
	"javascript-typescript",
	"python",
	"ruby",
	"swift",
}

func Execute(ctx context.Context, client githubapi.RESTClient, input Input) (Output, error) {
	languages, err := Validate(input)
	if err != nil {
		return Output{}, err
	}

	path := defaultSetupPath(input.Owner, input.Repo)

	var current CurrentConfig
	if err := client.DoWithContext(ctx, http.MethodGet, path, nil, &current); err != nil {
		return Output{}, err
	}
	current.Languages = normalizeLanguageSlice(current.Languages)

	desired := DesiredConfig{
		State:       "configured",
		Languages:   languages,
		RunnerType:  "standard",
		RunnerLabel: nil,
		QuerySuite:  "default",
		ThreatModel: "remote",
	}

	output := Output{
		Owner:   input.Owner,
		Repo:    input.Repo,
		Before:  current,
		After:   desired,
		Changed: false,
	}

	if configMatches(current, desired) {
		return output, nil
	}

	request := patchRequest{
		State:       desired.State,
		Languages:   desired.Languages,
		RunnerType:  desired.RunnerType,
		QuerySuite:  desired.QuerySuite,
		ThreatModel: desired.ThreatModel,
	}
	body, err := json.Marshal(request)
	if err != nil {
		return Output{}, err
	}

	var patched patchResponse
	if err := client.DoWithContext(ctx, http.MethodPatch, path, bytes.NewReader(body), &patched); err != nil {
		return Output{}, err
	}

	output.Changed = true
	output.RunID = patched.RunID
	output.RunURL = patched.RunURL
	return output, nil
}

func Validate(input Input) ([]string, error) {
	if input.Owner == "" {
		return nil, validation.New("owner is required")
	}
	if input.Repo == "" {
		return nil, validation.New("repo is required")
	}
	if strings.Contains(input.Repo, "/") {
		return nil, validation.New("repo must not contain /")
	}
	if input.Languages == "" {
		return nil, validation.New("languages is required")
	}

	languages, err := NormalizeLanguages(input.Languages)
	if err != nil {
		return nil, err
	}
	return languages, nil
}

func NormalizeLanguages(raw string) ([]string, error) {
	parts := strings.Split(raw, ",")
	languages := make(map[string]struct{})

	for _, part := range parts {
		language := strings.TrimSpace(part)
		if language == "" {
			return nil, validation.New("languages must not contain empty values")
		}
		if _, ok := allowedLanguages[language]; !ok {
			return nil, validation.New("languages must contain only: " + strings.Join(allowedLanguageList, ", "))
		}
		languages[language] = struct{}{}
	}

	normalized := make([]string, 0, len(languages))
	for language := range languages {
		normalized = append(normalized, language)
	}
	sort.Strings(normalized)

	return normalized, nil
}

func defaultSetupPath(owner string, repo string) string {
	return fmt.Sprintf(
		"repos/%s/%s/code-scanning/default-setup",
		url.PathEscape(owner),
		url.PathEscape(repo),
	)
}

func configMatches(current CurrentConfig, desired DesiredConfig) bool {
	return current.State == desired.State &&
		equalStringSlices(current.Languages, desired.Languages) &&
		runnerTypeMatches(current.RunnerType, desired.RunnerType) &&
		current.QuerySuite == desired.QuerySuite &&
		current.ThreatModel == desired.ThreatModel
}

func runnerTypeMatches(current *string, desired string) bool {
	return current != nil && *current == desired
}

func normalizeLanguageSlice(languages []string) []string {
	if len(languages) == 0 {
		return []string{}
	}

	seen := make(map[string]struct{}, len(languages))
	for _, language := range languages {
		seen[language] = struct{}{}
	}

	normalized := make([]string, 0, len(seen))
	for language := range seen {
		normalized = append(normalized, language)
	}
	sort.Strings(normalized)
	return normalized
}

func equalStringSlices(left []string, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}
