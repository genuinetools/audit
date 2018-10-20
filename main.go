package main

import (
	"context"
	"encoding/base64"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"golang.org/x/oauth2"

	"github.com/genuinetools/audit/version"
	"github.com/genuinetools/pkg/cli"
	"github.com/google/go-github/github"
	"github.com/sirupsen/logrus"
)

var (
	token string
	orgs  stringSlice
	repo  string
	owner bool

	debug bool
)

// stringSlice is a slice of strings
type stringSlice []string

// implement the flag interface for stringSlice
func (s *stringSlice) String() string {
	return fmt.Sprintf("%s", *s)
}
func (s *stringSlice) Set(value string) error {
	*s = append(*s, value)
	return nil
}

func main() {
	// Create a new cli program.
	p := cli.NewProgram()
	p.Name = "audit"
	p.Description = "Tool to audit what collaborators, hooks, and deploy keys are on your GitHub repositories"

	// Set the GitCommit and Version.
	p.GitCommit = version.GITCOMMIT
	p.Version = version.VERSION

	// Setup the global flags.
	p.FlagSet = flag.NewFlagSet("global", flag.ExitOnError)
	p.FlagSet.StringVar(&token, "token", os.Getenv("GITHUB_TOKEN"), "GitHub API token (or env var GITHUB_TOKEN)")
	p.FlagSet.Var(&orgs, "orgs", "specific orgs to check (e.g. 'genuinetools')")
	p.FlagSet.StringVar(&repo, "repo", "", "specific repo to test (e.g. 'genuinetools/audit')")
	p.FlagSet.BoolVar(&owner, "owner", false, "only audit repos the token owner owns")
	p.FlagSet.BoolVar(&debug, "d", false, "enable debug logging")
	p.FlagSet.BoolVar(&debug, "debug", false, "enable debug logging")

	// Set the before function.
	p.Before = func(ctx context.Context) error {
		// Set the log level.
		if debug {
			logrus.SetLevel(logrus.DebugLevel)
		}

		if token == "" {
			return errors.New("GitHub token cannot be empty")
		}

		if owner && len(orgs) > 0 {
			return errors.New("Cannot filter by organization while restricting to repos the token owner owns")
		}

		return nil
	}

	// Set the main program action.
	p.Action = func(ctx context.Context, args []string) error {
		// On ^C, or SIGTERM handle exit.
		signals := make(chan os.Signal, 0)
		signal.Notify(signals, os.Interrupt)
		signal.Notify(signals, syscall.SIGTERM)
		var cancel context.CancelFunc
		ctx, cancel = context.WithCancel(ctx)
		go func() {
			for sig := range signals {
				cancel()
				logrus.Infof("Received %s, exiting.", sig.String())
				os.Exit(0)
			}
		}()

		// Create the http client.
		ts := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: token},
		)
		tc := oauth2.NewClient(ctx, ts)

		// Create the github rest client.
		restClient := github.NewClient(tc)

		// Create the github graphql client.
		// Create a graphql client
		graphqlClient := NewGQLClient("https://api.github.com/graphql", map[string]string{
			"Authorization": "bearer " + token,
		})

		logrus.Debug("Getting current user...")
		// Get the current user
		var respData loginData
		if err := graphqlClient.Execute(GQLRequest{
			Query: queryGetLogin,
		}, &respData, nil); err != nil {
			return fmt.Errorf("Getting user failed: %v", err)
		}
		username := respData["viewer"]["login"]
		logrus.Debugf("current user is %s", username)

		var affiliations []string
		if owner {
			affiliations = []string{"OWNER"}
		} else {
			affiliations = []string{"OWNER", "COLLABORATOR", "ORGANIZATION_MEMBER"}
		}
		logrus.Debugf("Setting affiliations to %s", strings.Join(affiliations, ","))

		if len(orgs) > 0 {
			// get repos for each org
			for _, org := range orgs {
				logrus.Debugf("Getting repositories for org %s...", org)
				err := getRepositories(ctx, restClient, graphqlClient, affiliations, repo, org, "", true)
				if err != nil {
					return err
				}
			}
		} else {
			// get repos for the user only
			logrus.Debugf("Getting repositories for user %s...", username)
			err := getRepositories(ctx, restClient, graphqlClient, affiliations, repo, username, "", false)
			if err != nil {
				return err
			}

		}
		return nil
	}

	// Run our program.
	p.Run()
}

func getRepositories(ctx context.Context, restClient *github.Client, graphqlClient *GQLClient, affiliations []string, searchRepo string, login string, cursor string, isOrg bool) error {

	var (
		repos       []ghrepo
		errors      []GQLError
		hasNextPage bool
		variables   = map[string]interface{}{
			"login":        login,
			"affiliations": affiliations,
		}
	)

	if len(searchRepo) < 1 {
		if len(cursor) > 0 {
			variables["cursor"] = cursor
			logrus.Debugf("Cursor set at %s", cursor)
		}

		// get repositories for the user or org
		if isOrg {
			logrus.Debugf("Executing GraphQL query to fetch repos under org %s", login)
			var data orgReposResponse

			if err := graphqlClient.Execute(GQLRequest{
				Query:     buildGetReposQuery("organization"),
				Variables: variables,
			}, &data, &errors); err != nil {
				return err
			}

			if data.Org.Repositories.PageInfo.HasNextPage {
				logrus.Debug("Setting next page true and end cursor")
				hasNextPage = true
				cursor = data.Org.Repositories.PageInfo.EndCursor
			}

			repos = data.Org.Repositories.Nodes
		} else {
			logrus.Debugf("Executing GraphQL query to fetch repos under user %s", login)
			var data userReposResponse
			if err := graphqlClient.Execute(GQLRequest{
				Query:     buildGetReposQuery("user"),
				Variables: variables,
			}, &data, &errors); err != nil {
				return err
			}

			if data.User.Repositories.PageInfo.HasNextPage {
				hasNextPage = true
				cursor = data.User.Repositories.PageInfo.EndCursor
			}

			repos = data.User.Repositories.Nodes
		}

	} else {
		logrus.Debugf("Executing GraphQL query to fetch only 1 repo: %s", searchRepo)
		var data repoResponse
		var errors []GQLError

		// get only one repo
		search := strings.SplitN(searchRepo, "/", 2)
		if err := graphqlClient.Execute(GQLRequest{
			Query: queryGetRepo,
			Variables: map[string]interface{}{
				"owner": search[0],
				"name":  search[1],
			},
		}, &data, &errors); err != nil {
			return err
		}

		repos = []ghrepo{data.Repository}
	}

	// handle each repo
	for _, repo := range repos {
		logrus.Debugf("Handling repo %s...", repo.Name)
		if err := handleRepo(ctx, restClient, repo); err != nil {
			logrus.WithError(err).Errorf("auditing %s failed", repo.NameWithOwner)
		}
	}

	if hasNextPage {
		return getRepositories(ctx, restClient, graphqlClient, affiliations, searchRepo, login, cursor, isOrg)
	}

	return nil
}

// handleRepo will return nil error if the user does not have access to something.
func handleRepo(ctx context.Context, restClient *github.Client, repo ghrepo) error {
	opt := &github.ListOptions{
		PerPage: 100,
	}

	logrus.Debugf("Executing REST query to list teams for %s", repo.NameWithOwner)
	teams, resp, err := restClient.Repositories.ListTeams(ctx, repo.Owner.Login, repo.Name, opt)
	if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusForbidden || err != nil {
		if _, ok := err.(*github.RateLimitError); ok {
			return err
		}

		return nil
	}
	if err != nil {
		return err
	}

	logrus.Debugf("Executing REST query to list hooks for %s", repo.NameWithOwner)
	hooks, resp, err := restClient.Repositories.ListHooks(ctx, repo.Owner.Login, repo.Name, opt)
	if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusForbidden || err != nil {
		if _, ok := err.(*github.RateLimitError); ok {
			return err
		}

		return nil
	}
	if err != nil {
		return err
	}

	// only print whole status if we have more that one collaborator
	if repo.Collaborators.TotalCount <= 1 && repo.DeployKeys.TotalCount < 1 && len(hooks) < 1 && repo.BranchProtectionRules.TotalCount < 1 && repo.Refs.TotalCount < 1 {
		return nil
	}

	output := fmt.Sprintf("%s -> \n", repo.NameWithOwner)

	if repo.Collaborators.TotalCount > 1 {
		push := []string{}
		pull := []string{}
		admin := []string{}
		logrus.Debugf("Executing REST query to check collaborators' team memberships for %s", repo.NameWithOwner)
		for _, c := range repo.Collaborators.Edges {
			userTeams := []github.Team{}
			for _, t := range teams {
				isMember, resp, err := restClient.Teams.GetTeamMembership(ctx, t.GetID(), c.Node.Login)
				if resp.StatusCode != http.StatusNotFound && resp.StatusCode != http.StatusForbidden && err == nil && isMember.GetState() == "active" {
					userTeams = append(userTeams, *t)
				}
			}

			switch c.Permission {
			case "ADMIN":
				permTeams := []string{}
				for _, t := range userTeams {
					if t.GetPermission() == "admin" {
						permTeams = append(permTeams, t.GetName())
					}
				}
				admin = append(admin, fmt.Sprintf("\t\t\t%s (teams: %s)", c.Node.Login, strings.Join(permTeams, ", ")))
			case "WRITE":
				push = append(push, fmt.Sprintf("\t\t\t%s", c.Node.Login))
			case "READ":
				pull = append(pull, fmt.Sprintf("\t\t\t%s", c.Node.Login))
			}
		}
		output += fmt.Sprintf("\tCollaborators (%d):\n", repo.Collaborators.TotalCount)
		output += fmt.Sprintf("\t\tAdmin (%d):\n%s\n", len(admin), strings.Join(admin, "\n"))
		output += fmt.Sprintf("\t\tWrite (%d):\n%s\n", len(push), strings.Join(push, "\n"))
		output += fmt.Sprintf("\t\tRead (%d):\n%s\n", len(pull), strings.Join(pull, "\n"))
	}

	if repo.DeployKeys.TotalCount > 0 {
		kstr := []string{}
		for _, k := range repo.DeployKeys.Nodes {
			keyURL, err := buildDeployKeyURL(repo.Owner.Login, repo.Name, k.ID)
			if err != nil {
				kstr = append(kstr, fmt.Sprintf("\t\t%s - ro:%t", k.Title, k.ReadOnly))
			} else {
				kstr = append(kstr, fmt.Sprintf("\t\t%s - ro:%t (%s)", k.Title, k.ReadOnly, keyURL))
			}
		}
		output += fmt.Sprintf("\tKeys (%d):\n%s\n", repo.DeployKeys.TotalCount, strings.Join(kstr, "\n"))
	}

	if len(hooks) > 0 {
		hstr := []string{}
		for _, h := range hooks {
			hstr = append(hstr, fmt.Sprintf("\t\t%s - active:%t (%s)", h.GetName(), h.GetActive(), h.GetURL()))
		}
		output += fmt.Sprintf("\tHooks (%d):\n%s\n", len(hstr), strings.Join(hstr, "\n"))
	}

	if repo.BranchProtectionRules.TotalCount > 0 {
		protectedBranches := []string{}
		for _, r := range repo.BranchProtectionRules.Nodes {
			protectedBranches = append(protectedBranches, r.Pattern)
		}
		output += fmt.Sprintf("\tProtected Branches (%d): %s\n", len(protectedBranches), strings.Join(protectedBranches, ", "))
	}

	if repo.Refs.TotalCount > 0 {
		unprotectedBranches := []string{}
		for _, r := range repo.Refs.Nodes {
			unprotectedBranches = append(unprotectedBranches, r.Name)
		}
		output += fmt.Sprintf("\tUnprotected Branches (%d): %s\n", len(unprotectedBranches), strings.Join(unprotectedBranches, ", "))
	}

	mergeMethods := "\tMerge Methods:"
	if repo.MergeCommitAllowed {
		mergeMethods += " mergeCommit"
	}
	if repo.SquashMergeAllowed {
		mergeMethods += " squash"
	}
	if repo.RebaseMergeAllowed {
		mergeMethods += " rebase"
	}
	output += mergeMethods + "\n"

	logrus.Debugf("Printing details for %s", repo.NameWithOwner)

	fmt.Printf("%s--\n\n", output)

	return nil
}

func in(a stringSlice, s string) bool {
	for _, b := range a {
		if b == s {
			return true
		}
	}
	return false
}

func usageAndExit(message string, exitCode int) {
	if message != "" {
		fmt.Fprintf(os.Stderr, message)
		fmt.Fprintf(os.Stderr, "\n\n")
	}
	flag.Usage()
	fmt.Fprintf(os.Stderr, "\n")
	os.Exit(exitCode)
}

func buildDeployKeyURL(owner, name, id string) (string, error) {
	decodedID, err := base64.StdEncoding.DecodeString(id)
	if err != nil {
		return "", err
	}
	keyID := strings.TrimPrefix(string(decodedID), "09:PublicKey")
	return fmt.Sprintf("https://api.github.com/repos/%s/%s/keys/%s", owner, name, keyID), nil
}
