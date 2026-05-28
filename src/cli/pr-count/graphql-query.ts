export const GRAPHQL_QUERY = `query ($owner: String!, $name: String!, $state: [PullRequestState!]) {
        repository(owner: $owner, name: $name) {
            pullRequests(states: $state) {
                totalCount
            }
        }
    }`;
