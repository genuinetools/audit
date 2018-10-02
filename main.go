package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

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

		// Create the github client.
		client := github.NewClient(tc)

		// If no organizations provided, audit repositories belonging to the current user.
		if len(orgs) == 0 {
			// Get the current user
			user, _, err := client.Users.Get(ctx, "")
			if err != nil {
				if v, ok := err.(*github.RateLimitError); ok {
					return fmt.Errorf("%s Limit: %d; Remaining: %d; Retry After: %s", v.Message, v.Rate.Limit, v.Rate.Remaining, time.Until(v.Rate.Reset.Time).String())
				}

				return fmt.Errorf("Getting user failed: %v", err)
			}
			username := *user.Login
			// add the current user to orgs
			orgs = append(orgs, username)
		}

		page := 1
		perPage := 100
		var affiliation string
		if owner {
			affiliation = "owner"
		} else {
			affiliation = "owner,collaborator,organization_member"
		}
		logrus.Debugf("Getting repositories...")
		if err := getRepositories(ctx, client, page, perPage, affiliation, repo, orgs); err != nil {
			if v, ok := err.(*github.RateLimitError); ok {
				logrus.Fatalf("%s Limit: %d; Remaining: %d; Retry After: %s", v.Message, v.Rate.Limit, v.Rate.Remaining, time.Until(v.Rate.Reset.Time).String())
			}

			logrus.Fatal(err)
		}
		return nil
	}

	// Run our program.
	p.Run()
}

func getRepositories(ctx context.Context, client *github.Client, page, perPage int, affiliation string, searchRepo string, orgs stringSlice) error {

	var (
		repos []*github.Repository
		resp  *github.Response
		err   error
	)
	if len(searchRepo) < 1 {
		// Get all the repos.
		repos, resp, err = client.Repositories.List(ctx, "", &github.RepositoryListOptions{
			Affiliation: affiliation,
			ListOptions: github.ListOptions{
				Page:    page,
				PerPage: perPage,
			},
		})
		if err != nil {
			return err
		}
	} else {
		// Find the one repo.
		repos, err = searchRepos(ctx, client, searchRepo)
	}
	if err != nil {
		return err
	}

	for _, repo := range repos {
		if !in(orgs, *repo.Owner.Login) {
			continue
		}
		logrus.Debugf("Handling repo %s...", repo.GetFullName())
		if err := handleRepo(ctx, client, repo); err != nil {
			if len(searchRepo) > 0 {
				return err
			}

			logrus.Warn(err)
		}
	}

	// Return early if we are on the last page.
	if resp == nil || page == resp.LastPage || resp.NextPage == 0 {
		return nil
	}

	page = resp.NextPage
	return getRepositories(ctx, client, page, perPage, affiliation, searchRepo, orgs)
}

func searchRepos(ctx context.Context, client *github.Client, searchRepo string) ([]*github.Repository, error) {
	optSearch := &github.SearchOptions{
		Sort:  "forks",
		Order: "desc",
		ListOptions: github.ListOptions{
			Page:    1,
			PerPage: 1,
		},
	}

	search := strings.SplitN(searchRepo, "/", 2)
	repos, _, err := client.Search.Repositories(ctx, fmt.Sprintf("org:%s in:name %s fork:true", search[0], search[1]), optSearch)
	if err != nil {
		return nil, err
	}

	if len(repos.Repositories) < 1 {
		return nil, fmt.Errorf("found no repositories matching: %s", searchRepo)
	}

	r := []*github.Repository{}
	for _, repo := range repos.Repositories {
		r = append(r, &repo)
	}
	return r, nil
}

// handleRepo will return nil error if the user does not have access to something.
func handleRepo(ctx context.Context, client *github.Client, repo *github.Repository) error {
	opt := &github.ListOptions{
		PerPage: 100,
	}

	teams, resp, err := client.Repositories.ListTeams(ctx, repo.GetOwner().GetLogin(), repo.GetName(), opt)
	if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusForbidden || err != nil {
		if _, ok := err.(*github.RateLimitError); ok {
			return err
		}

		return nil
	}
	if err != nil {
		return err
	}

	collabs, resp, err := client.Repositories.ListCollaborators(ctx, repo.GetOwner().GetLogin(), repo.GetName(), &github.ListCollaboratorsOptions{ListOptions: *opt})
	if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusForbidden || err != nil {
		if _, ok := err.(*github.RateLimitError); ok {
			return err
		}

		return nil
	}
	if err != nil {
		return err
	}

	keys, resp, err := client.Repositories.ListKeys(ctx, repo.GetOwner().GetLogin(), repo.GetName(), opt)
	if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusForbidden || err != nil {
		if _, ok := err.(*github.RateLimitError); ok {
			return err
		}

		return nil
	}
	if err != nil {
		return err
	}

	hooks, resp, err := client.Repositories.ListHooks(ctx, repo.GetOwner().GetLogin(), repo.GetName(), opt)
	if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusForbidden || err != nil {
		if _, ok := err.(*github.RateLimitError); ok {
			return err
		}

		return nil
	}
	if err != nil {
		return err
	}

	branches, _, err := client.Repositories.ListBranches(ctx, repo.GetOwner().GetLogin(), repo.GetName(), opt)
	if err != nil {
		return err
	}
	protectedBranches := []string{}
	unprotectedBranches := []string{}
	for _, branch := range branches {
		// we must get the individual branch for the branch protection to work
		b, _, err := client.Repositories.GetBranch(ctx, repo.GetOwner().GetLogin(), repo.GetName(), branch.GetName())
		if err != nil {
			return err
		}
		if b.GetProtected() {
			protectedBranches = append(protectedBranches, b.GetName())
		} else {
			unprotectedBranches = append(unprotectedBranches, b.GetName())
		}
	}

	// only print whole status if we have more that one collaborator
	if len(collabs) <= 1 && len(keys) < 1 && len(hooks) < 1 && len(protectedBranches) < 1 && len(unprotectedBranches) < 1 {
		return nil
	}

	output := fmt.Sprintf("%s -> \n", repo.GetFullName())

	if len(collabs) > 1 {
		push := []string{}
		pull := []string{}
		admin := []string{}
		for _, c := range collabs {
			userTeams := []github.Team{}
			for _, t := range teams {
				isMember, resp, err := client.Teams.GetTeamMembership(ctx, t.GetID(), c.GetLogin())
				if resp.StatusCode != http.StatusNotFound && resp.StatusCode != http.StatusForbidden && err == nil && isMember.GetState() == "active" {
					userTeams = append(userTeams, *t)
				}
			}

			perms := c.GetPermissions()

			switch {
			case perms["admin"]:
				permTeams := []string{}
				for _, t := range userTeams {
					if t.GetPermission() == "admin" {
						permTeams = append(permTeams, t.GetName())
					}
				}
				admin = append(admin, fmt.Sprintf("\t\t\t%s (teams: %s)", c.GetLogin(), strings.Join(permTeams, ", ")))
			case perms["push"]:
				push = append(push, fmt.Sprintf("\t\t\t%s", c.GetLogin()))
			case perms["pull"]:
				pull = append(pull, fmt.Sprintf("\t\t\t%s", c.GetLogin()))
			}
		}
		output += fmt.Sprintf("\tCollaborators (%d):\n", len(collabs))
		output += fmt.Sprintf("\t\tAdmin (%d):\n%s\n", len(admin), strings.Join(admin, "\n"))
		output += fmt.Sprintf("\t\tWrite (%d):\n%s\n", len(push), strings.Join(push, "\n"))
		output += fmt.Sprintf("\t\tRead (%d):\n%s\n", len(pull), strings.Join(pull, "\n"))
	}

	if len(keys) > 0 {
		kstr := []string{}
		for _, k := range keys {
			kstr = append(kstr, fmt.Sprintf("\t\t%s - ro:%t (%s)", k.GetTitle(), k.GetReadOnly(), k.GetURL()))
		}
		output += fmt.Sprintf("\tKeys (%d):\n%s\n", len(kstr), strings.Join(kstr, "\n"))
	}

	if len(hooks) > 0 {
		hstr := []string{}
		for _, h := range hooks {
			hstr = append(hstr, fmt.Sprintf("\t\t%s - active:%t (%s)", h.GetName(), h.GetActive(), h.GetURL()))
		}
		output += fmt.Sprintf("\tHooks (%d):\n%s\n", len(hstr), strings.Join(hstr, "\n"))
	}

	if len(protectedBranches) > 0 {
		output += fmt.Sprintf("\tProtected Branches (%d): %s\n", len(protectedBranches), strings.Join(protectedBranches, ", "))
	}

	if len(unprotectedBranches) > 0 {
		output += fmt.Sprintf("\tUnprotected Branches (%d): %s\n", len(unprotectedBranches), strings.Join(unprotectedBranches, ", "))
	}

	repo, _, err = client.Repositories.Get(ctx, repo.GetOwner().GetLogin(), repo.GetName())
	if err != nil {
		return err
	}

	mergeMethods := "\tMerge Methods:"
	if repo.GetAllowMergeCommit() {
		mergeMethods += " mergeCommit"
	}
	if repo.GetAllowSquashMerge() {
		mergeMethods += " squash"
	}
	if repo.GetAllowRebaseMerge() {
		mergeMethods += " rebase"
	}
	output += mergeMethods + "\n"

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
