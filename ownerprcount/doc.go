// Package ownerprcount aggregates open pull request counts for every repository
// owned by a GitHub user or organization that is visible to the authenticated
// client.
//
// Execute follows the complete repositories connection. Consequently, a result
// describes values observed over the duration of the requests, not an atomic
// snapshot of GitHub at one instant.
package ownerprcount
