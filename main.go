package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"golang.org/x/oauth2"

	"github.com/google/go-github/github"
	"github.com/jessfraz/audit/version"
	"github.com/sirupsen/logrus"
)

const (
	// BANNER is what is printed for help/info output.
	BANNER = `                 _ _ _
  __ _ _   _  __| (_) |_
 / _` + "`" + ` | | | |/ _` + "`" + ` | | __|
| (_| | |_| | (_| | | |_
 \__,_|\__,_|\__,_|_|\__|

 Auditing what collaborators, hooks, and deploy keys you have added on all your GitHub repositories.
 Version: %s
 Build: %s

`
)

var (
	token string

	debug bool
	vrsn  bool
	owner bool
)

func init() {
	// parse flags
	flag.StringVar(&token, "token", os.Getenv("GITHUB_TOKEN"), "GitHub API token (or env var GITHUB_TOKEN)")

	flag.BoolVar(&vrsn, "version", false, "print version and exit")
	flag.BoolVar(&vrsn, "v", false, "print version and exit (shorthand)")
	flag.BoolVar(&debug, "d", false, "run in debug mode")
	flag.BoolVar(&owner, "owner", false, "only audit repos the token owner owns")

	flag.Usage = func() {
		fmt.Fprint(os.Stderr, fmt.Sprintf(BANNER, version.VERSION, version.GITCOMMIT))
		flag.PrintDefaults()
	}

	flag.Parse()

	if vrsn {
		fmt.Printf("audit version %s, build %s", version.VERSION, version.GITCOMMIT)
		os.Exit(0)
	}

	// set log level
	if debug {
		logrus.SetLevel(logrus.DebugLevel)
	}

	if token == "" {
		usageAndExit("GitHub token cannot be empty.", 1)
	}
}

func main() {
	// On ^C, or SIGTERM handle exit.
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	signal.Notify(c, syscall.SIGTERM)
	go func() {
		for sig := range c {
			logrus.Infof("Received %s, exiting.", sig.String())
			os.Exit(0)
		}
	}()

	ctx := context.Background()

	// Create the http client.
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)

	// Create the github client.
	client := github.NewClient(tc)
	page := 1
	perPage := 100
	var affiliation string
	if owner {
		affiliation = "owner"
	} else {
		affiliation = "owner,collaborator,organization_member"
	}
	logrus.Debugf("Getting repositories...")
	if err := getRepositories(ctx, client, page, perPage, affiliation); err != nil {
		logrus.Fatal(err)
	}
}

func getRepositories(ctx context.Context, client *github.Client, page, perPage int, affiliation string) error {
	opt := &github.RepositoryListOptions{
		Affiliation: affiliation,
		ListOptions: github.ListOptions{
			Page:    page,
			PerPage: perPage,
		},
	}
	repos, resp, err := client.Repositories.List(ctx, "", opt)
	if err != nil {
		return err
	}

	for _, repo := range repos {
		logrus.Debugf("Handling repo %s...", *repo.FullName)
		if err := handleRepo(ctx, client, repo); err != nil {
			logrus.Warn(err)
		}
	}

	// Return early if we are on the last page.
	if page == resp.LastPage || resp.NextPage == 0 {
		return nil
	}

	page = resp.NextPage
	return getRepositories(ctx, client, page, perPage, affiliation)
}

// handleRepo will return nil error if the user does not have access to something.
func handleRepo(ctx context.Context, client *github.Client, repo *github.Repository) error {
	opt := &github.ListOptions{
		PerPage: 100,
	}

	teams, resp, err := client.Repositories.ListTeams(ctx, *repo.Owner.Login, *repo.Name, opt)
	if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusForbidden {
		return nil
	}
	if err != nil {
		return err
	}

	collabs, resp, err := client.Repositories.ListCollaborators(ctx, *repo.Owner.Login, *repo.Name, &github.ListCollaboratorsOptions{ListOptions: *opt})
	if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusForbidden {
		return nil
	}
	if err != nil {
		return err
	}

	keys, resp, err := client.Repositories.ListKeys(ctx, *repo.Owner.Login, *repo.Name, opt)
	if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusForbidden {
		return nil
	}
	if err != nil {
		return err
	}

	hooks, resp, err := client.Repositories.ListHooks(ctx, *repo.Owner.Login, *repo.Name, opt)
	if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusForbidden {
		return nil
	}
	if err != nil {
		return err
	}

	branches, _, err := client.Repositories.ListBranches(ctx, *repo.Owner.Login, *repo.Name, opt)
	if err != nil {
		return err
	}
	protectedBranches := []string{}
	unprotectedBranches := []string{}
	for _, branch := range branches {
		// we must get the individual branch for the branch protection to work
		b, _, err := client.Repositories.GetBranch(ctx, *repo.Owner.Login, *repo.Name, branch.GetName())
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

	output := fmt.Sprintf("%s -> \n", *repo.FullName)

	if len(collabs) > 1 {
		push := []string{}
		pull := []string{}
		admin := []string{}
		for _, c := range collabs {
			userTeams := []github.Team{}
			for _, t := range teams {
				isMember, resp, err := client.Organizations.GetTeamMembership(ctx, *t.ID, *c.Login)
				if resp.StatusCode != http.StatusNotFound && resp.StatusCode != http.StatusForbidden && err == nil && *isMember.State == "active" {
					userTeams = append(userTeams, *t)
				}
			}

			perms := *c.Permissions

			switch {
			case perms["admin"]:
				permTeams := []string{}
				for _, t := range userTeams {
					if *t.Permission == "admin" {
						permTeams = append(permTeams, *t.Name)
					}
				}
				admin = append(admin, fmt.Sprintf("\t\t\t%s (teams: %s)", *c.Login, strings.Join(permTeams, ", ")))
			case perms["push"]:
				push = append(push, fmt.Sprintf("\t\t\t%s", *c.Login))
			case perms["pull"]:
				pull = append(pull, fmt.Sprintf("\t\t\t%s", *c.Login))
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
			kstr = append(kstr, fmt.Sprintf("\t\t%s - ro:%t (%s)", *k.Title, *k.ReadOnly, *k.URL))
		}
		output += fmt.Sprintf("\tKeys (%d):\n%s\n", len(kstr), strings.Join(kstr, "\n"))
	}

	if len(hooks) > 0 {
		hstr := []string{}
		for _, h := range hooks {
			hstr = append(hstr, fmt.Sprintf("\t\t%s - active:%t (%s)", *h.Name, *h.Active, *h.URL))
		}
		output += fmt.Sprintf("\tHooks (%d):\n%s\n", len(hstr), strings.Join(hstr, "\n"))
	}

	if len(protectedBranches) > 0 {
		output += fmt.Sprintf("\tProtected Branches (%d): %s\n", len(protectedBranches), strings.Join(protectedBranches, ", "))
	}

	if len(unprotectedBranches) > 0 {
		output += fmt.Sprintf("\tUnprotected Branches (%d): %s\n", len(unprotectedBranches), strings.Join(unprotectedBranches, ", "))
	}

	fmt.Printf("%s--\n\n", output)

	return nil
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
