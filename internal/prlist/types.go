package prlist

type Input struct {
	Owner string
	Name  string
	State *string
	After *string
	First *int
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
	Nodes    []PullRequestNode `json:"nodes"`
	PageInfo PageInfo          `json:"pageInfo"`
}

type PullRequestNode struct {
	Number         int            `json:"number"`
	CreatedAt      string         `json:"createdAt"`
	State          string         `json:"state"`
	MergedAt       *string        `json:"mergedAt"`
	ChangedFiles   int            `json:"changedFiles"`
	ReviewDecision *string        `json:"reviewDecision"`
	Comments       Comments       `json:"comments"`
	Author         *Author        `json:"author"`
	ReviewRequests ReviewRequests `json:"reviewRequests"`
	Reviews        Reviews        `json:"reviews"`
}

type Comments struct {
	TotalCount int `json:"totalCount"`
}

type Author struct {
	Login string `json:"login"`
}

type ReviewRequests struct {
	Nodes []ReviewRequestNode `json:"nodes"`
}

type ReviewRequestNode struct {
	RequestedReviewer *RequestedReviewer `json:"requestedReviewer"`
}

type RequestedReviewer struct {
	Login string `json:"login,omitempty"`
	Slug  string `json:"slug,omitempty"`
}

type Reviews struct {
	Nodes []ReviewNode `json:"nodes"`
}

type ReviewNode struct {
	Author *Author `json:"author"`
}

type PageInfo struct {
	HasNextPage bool    `json:"hasNextPage"`
	EndCursor   *string `json:"endCursor"`
}
