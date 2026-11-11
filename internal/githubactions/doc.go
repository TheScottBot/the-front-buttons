// Package githubactions is the GitHub Actions client behind the pipeline provider
// port.
//
// It is an adapter because it talks to the network, so only the composition root
// constructs it and bricks see only the port.
package githubactions
