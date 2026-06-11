package pullrequestcreationpolicy

type Input struct {
	Owner  string
	Repo   string
	Policy string
}

type PolicyConfig struct {
	PullRequestCreationPolicy string `json:"pull_request_creation_policy"`
}

type Output struct {
	Owner   string       `json:"owner"`
	Repo    string       `json:"repo"`
	Changed bool         `json:"changed"`
	Before  PolicyConfig `json:"before"`
	After   PolicyConfig `json:"after"`
}

type patchRequest struct {
	PullRequestCreationPolicy string `json:"pull_request_creation_policy"`
}
