package prcount

const graphQLQuery = `query ($owner: String!, $name: String!, $state: [PullRequestState!]) {
        repository(owner: $owner, name: $name) {
            pullRequests(states: $state) {
                totalCount
            }
        }
    }`
