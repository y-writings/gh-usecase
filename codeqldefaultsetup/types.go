package codeqldefaultsetup

import (
	"context"
	"io"
)

type Client interface {
	DoWithContext(ctx context.Context, method string, path string, body io.Reader, response interface{}) error
}

type Input struct {
	Owner     string
	Repo      string
	Languages []string
}

type CurrentConfig struct {
	State       string   `json:"state"`
	Languages   []string `json:"languages"`
	RunnerType  *string  `json:"runner_type"`
	RunnerLabel *string  `json:"runner_label"`
	QuerySuite  string   `json:"query_suite"`
	ThreatModel string   `json:"threat_model"`
	Schedule    *string  `json:"schedule"`
	UpdatedAt   *string  `json:"updated_at"`
}

type DesiredConfig struct {
	State       string   `json:"state"`
	Languages   []string `json:"languages"`
	RunnerType  string   `json:"runner_type"`
	RunnerLabel *string  `json:"runner_label"`
	QuerySuite  string   `json:"query_suite"`
	ThreatModel string   `json:"threat_model"`
}

type Output struct {
	Owner   string        `json:"owner"`
	Repo    string        `json:"repo"`
	Changed bool          `json:"changed"`
	Before  CurrentConfig `json:"before"`
	After   DesiredConfig `json:"after"`
	RunID   *int64        `json:"run_id"`
	RunURL  *string       `json:"run_url"`
}

type patchRequest struct {
	State       string   `json:"state"`
	Languages   []string `json:"languages"`
	RunnerType  string   `json:"runner_type"`
	QuerySuite  string   `json:"query_suite"`
	ThreatModel string   `json:"threat_model"`
}

type patchResponse struct {
	RunID  *int64  `json:"run_id"`
	RunURL *string `json:"run_url"`
}
