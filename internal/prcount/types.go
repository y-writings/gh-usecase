package prcount

type Input struct {
	Owner string
	Name  string
	State *string
}

type Output struct {
	Data Data `json:"data"`
}

type Data struct {
	Repository Repository `json:"repository"`
}

type Repository struct {
	PullRequests PullRequests `json:"pullRequests"`
}

type PullRequests struct {
	TotalCount int `json:"totalCount"`
}
