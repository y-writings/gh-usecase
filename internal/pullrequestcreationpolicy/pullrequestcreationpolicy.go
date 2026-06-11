package pullrequestcreationpolicy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/y-writings/gh-usecase/internal/githubapi"
	"github.com/y-writings/gh-usecase/internal/validation"
)

func Execute(ctx context.Context, client githubapi.RESTClient, input Input) (Output, error) {
	policy, err := Validate(input)
	if err != nil {
		return Output{}, err
	}

	path := repositoryPath(input.Owner, input.Repo)

	var current PolicyConfig
	if err := client.DoWithContext(ctx, http.MethodGet, path, nil, &current); err != nil {
		return Output{}, err
	}

	desired := PolicyConfig{PullRequestCreationPolicy: policy}
	output := Output{
		Owner:   input.Owner,
		Repo:    input.Repo,
		Changed: false,
		Before:  current,
		After:   desired,
	}

	if current.PullRequestCreationPolicy == policy {
		return output, nil
	}

	request := patchRequest{PullRequestCreationPolicy: policy}
	body, err := json.Marshal(request)
	if err != nil {
		return Output{}, err
	}

	var patched PolicyConfig
	if err := client.DoWithContext(ctx, http.MethodPatch, path, bytes.NewReader(body), &patched); err != nil {
		return Output{}, err
	}

	output.Changed = true
	return output, nil
}

func Validate(input Input) (string, error) {
	if input.Owner == "" {
		return "", validation.New("owner is required")
	}
	if input.Repo == "" {
		return "", validation.New("repo is required")
	}
	if strings.Contains(input.Repo, "/") {
		return "", validation.New("repo must not contain /")
	}
	if input.Policy == "" {
		return "", validation.New("policy is required")
	}
	if input.Policy != "all" && input.Policy != "collaborators_only" {
		return "", validation.New("policy must be all or collaborators_only")
	}
	return input.Policy, nil
}

func repositoryPath(owner string, repo string) string {
	return fmt.Sprintf(
		"repos/%s/%s",
		url.PathEscape(owner),
		url.PathEscape(repo),
	)
}
