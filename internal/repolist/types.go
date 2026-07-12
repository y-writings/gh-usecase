package repolist

type Input struct {
	Owner string
	First *int
	After *string
}

type Output struct {
	Data Data `json:"data"`
}

type Data struct {
	RepositoryOwner *RepositoryOwner `json:"repositoryOwner"`
}

type RepositoryOwner struct {
	Repositories Repositories `json:"repositories"`
}

type Repositories struct {
	Nodes    []RepositoryNode `json:"nodes"`
	PageInfo PageInfo         `json:"pageInfo"`
}

type RepositoryNode struct {
	Name          string `json:"name"`
	NameWithOwner string `json:"nameWithOwner"`
	URL           string `json:"url"`
}

type PageInfo struct {
	HasNextPage bool    `json:"hasNextPage"`
	EndCursor   *string `json:"endCursor"`
}
