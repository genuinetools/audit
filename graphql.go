package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

// GQLRequest is the GraphQL request containing Query and Variables
type GQLRequest struct {
	Query     string                 `json:"query"`
	Variables map[string]interface{} `json:"variables"`
}

// GQLResponse is the response from GraphQL server
type GQLResponse struct {
	Data   *json.RawMessage `json:"data"`
	Errors *json.RawMessage `json:"errors"`
}

// GQLError is a the GraphQL error from GitHub API
type GQLError struct {
	Message   string             `json:"message"`
	Locations []GQLErrorLocation `json:"locations"`
	Type      string             `json:"type"`
	Path      []interface{}      `json:"path"`
}

// Error returns the error message
func (e GQLError) Error() string {
	return e.Message
}

// GQLErrorLocation is the location of error in the query string
type GQLErrorLocation struct {
	Line   int `json:"line"`
	Column int `json:"column"`
}

// GQLClient can execute GraphQL queries against an endpoint
type GQLClient struct {
	Endpoint string
	Headers  map[string]string
	client   *http.Client
}

// NewGQLClient returns a GQLClient for given endpoint and headers
func NewGQLClient(endpoint string, headers map[string]string) *GQLClient {
	return &GQLClient{
		Endpoint: endpoint,
		Headers:  headers,
		client:   &http.Client{},
	}
}

// Execute executes the GQLRequest r using the GQLClient c and returns an error
// Response data and errors can be unmarshalled to the passed interfaces
func (c *GQLClient) Execute(r GQLRequest, data interface{}, errors interface{}) error {
	payload, err := json.Marshal(r)
	if err != nil {
		return err
	}
	req, err := http.NewRequest("POST", c.Endpoint, bytes.NewBuffer(payload))
	if err != nil {
		return err
	}
	for k, v := range c.Headers {
		req.Header.Set(k, v)
	}
	res, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	var response GQLResponse
	err = json.NewDecoder(res.Body).Decode(&response)
	if err != nil {
		return err
	}

	err = json.Unmarshal(*response.Data, data)
	if err != nil {
		return err
	}
	if response.Errors != nil {
		err = json.Unmarshal(*response.Errors, errors)
		if err != nil {
			return err
		}
	}

	return nil
}

// queryGetLogin is the GraphQL query to get login name of the user
const queryGetLogin = `
query {
  viewer {
    login
  }
}
`

// loginData is the response data for QUERY_GET_LOGIN
type loginData map[string]map[string]string

// buildGetReposQuery takes a param (user or organization) and returns the
// correct GraphQL query to fetch repositories under that resource
func buildGetReposQuery(param string) string {
	return fmt.Sprintf(`
        query getRepos(
          $login: String!,
          $affiliations: [RepositoryAffiliation]!,
          $cursor: String
        ) {
          %s (login: $login) {
            repositories(
              first: 100,
              affiliations: $affiliations,
              orderBy: {field: STARGAZERS, direction: DESC},
              after: $cursor
        ) {
              totalCount
              pageInfo {
                startCursor
                endCursor
                hasNextPage
                hasPreviousPage
              }
              nodes {
                owner {
                  login
                }
                name
                nameWithOwner
                stargazers {
                  totalCount
                }
                refs(first: 100, refPrefix: "refs/heads/") {
                  totalCount
                  nodes {
                    name
                  }
                }
                mergeCommitAllowed
                rebaseMergeAllowed
                squashMergeAllowed
                defaultBranchRef {
                  name
                }
                branchProtectionRules(first: 100) {
                  totalCount
                  nodes {
                    pattern
                  }
                }
                deployKeys(first: 100) {
                  totalCount
                  nodes {
                    id
                    title
                    readOnly
                  }
                }
                collaborators(first: 100) {
                  totalCount
                  edges {
                    permission
                    node {
                      login
                    }
                  }
                }
              }
            }
          }
        }
    `, param)
}

// queryGetRepo is the GraphQL query to get details about a repository
const queryGetRepo = `
query getRepo($owner: String!, $name: String!) {
  repository(owner: $owner, name: $name) {
    owner {
      login
    }
    name
    nameWithOwner
    stargazers {
      totalCount
    }
    mergeCommitAllowed
    rebaseMergeAllowed
    squashMergeAllowed
    defaultBranchRef {
      name
    }
    refs(first: 100, refPrefix: "refs/heads/") {
      totalCount
      nodes {
        name
      }
    }
    branchProtectionRules(first: 100) {
      totalCount
      nodes {
        pattern
      }
    }
    deployKeys(first: 100) {
      totalCount
      nodes {
        id
        title
        readOnly
      }
    }
    collaborators(first: 100) {
      totalCount
      edges {
        permission
        node {
          login
        }
      }
    }
  }
}

`

type userReposResponse struct {
	User repos `json:"user"`
}

type orgReposResponse struct {
	Org repos `json:"organization"`
}

type repos struct {
	Repositories repositoriesInfo `json:"repositories"`
}

type repositoriesInfo struct {
	TotalCount int      `json:"totalCount"`
	PageInfo   pageInfo `json:"pageInfo"`
	Nodes      []ghrepo `json:"nodes"`
}

type pageInfo struct {
	StartCursor string `json:"startCursor"`
	EndCursor   string `json:"endCursor"`
	HasNextPage bool   `json:"hasNextPage"`
}

type repoResponse struct {
	Repository ghrepo `json:"repository"`
}

type ghrepo struct {
	Name                  string           `json:"name"`
	Owner                 collaboratorNode `json:"owner"`
	NameWithOwner         string           `json:"nameWithOwner"`
	Stargazers            countNodeName    `json:"stargazers"`
	MergeCommitAllowed    bool             `json:"mergeCommitAllowed"`
	RebaseMergeAllowed    bool             `json:"rebaseMergeAllowed"`
	SquashMergeAllowed    bool             `json:"squashMergeAllowed"`
	Refs                  countNodeName    `json:"refs"`
	BranchProtectionRules countNodeName    `json:"branchProtectionRules"`
	DeployKeys            countNodeName    `json:"deployKeys"`
	Collaborators         collaborators    `json:"collaborators"`
}

type stargazers struct {
	TotalCount int `json:"totalCount"`
}

type countNodeName struct {
	TotalCount int           `json:"totalCount"`
	Nodes      []nodeElement `json:"nodes"`
}

type nodeElement struct {
	Name     string `json:"name"`
	Title    string `json:"title"`
	ReadOnly bool   `json:"readOnly"`
	ID       string `json:"id"`
	Pattern  string `json:"pattern"`
}

type collaborators struct {
	TotalCount int                `json:"totalCount"`
	Edges      []collaboratorEdge `json:"edges"`
}

type collaboratorEdge struct {
	Permission string           `json:"permission"`
	Node       collaboratorNode `json:"node"`
}

type collaboratorNode struct {
	Login string `json:"login"`
}
