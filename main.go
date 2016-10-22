package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"golang.org/x/oauth2"

	"github.com/Sirupsen/logrus"
	"github.com/google/go-github/github"
)

const (
	// BANNER is what is printed for help/info output.
	BANNER = "audit - %s\n"
	// VERSION is the binary version.
	VERSION = "v0.1.0"
)

var (
	token string

	debug   bool
	version bool
	owner   bool
)

func init() {
	// parse flags
	flag.StringVar(&token, "token", "", "GitHub API token")

	flag.BoolVar(&version, "version", false, "print version and exit")
	flag.BoolVar(&version, "v", false, "print version and exit (shorthand)")
	flag.BoolVar(&debug, "d", false, "run in debug mode")
	flag.BoolVar(&owner, "owner", false, "only audit repos the token owner owns")

	flag.Usage = func() {
		fmt.Fprint(os.Stderr, fmt.Sprintf(BANNER, VERSION))
		flag.PrintDefaults()
	}

	flag.Parse()

	if version {
		fmt.Printf("%s", VERSION)
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

	// Create the http client.
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(oauth2.NoContext, ts)

	// Create the github client.
	client := github.NewClient(tc)

	page := 1
	perPage := 20
	var affiliation string
	if owner {
		affiliation = "owner"
	} else {
		affiliation = "owner,collaborator,organization_member"
	}
	if err := getRepositories(client, page, perPage, affiliation); err != nil {
		logrus.Fatal(err)
	}
}

func getRepositories(client *github.Client, page, perPage int, affiliation string) error {
	opt := &github.RepositoryListOptions{
		Affiliation: affiliation,
		ListOptions: github.ListOptions{
			Page:    page,
			PerPage: perPage,
		},
	}
	repos, resp, err := client.Repositories.List("", opt)
	if err != nil {
		return err
	}

	for _, repo := range repos {
		if err := handleRepo(client, repo); err != nil {
			logrus.Warn(err)
		}
	}

	// Return early if we are on the last page.
	if page == resp.LastPage || resp.NextPage == 0 {
		return nil
	}

	page = resp.NextPage
	return getRepositories(client, page, perPage, affiliation)
}

// handleRepo will return nil error if the user does not have access to something.
func handleRepo(client *github.Client, repo *github.Repository) error {
	opt := &github.ListOptions{
		PerPage: 100,
	}
	collabs, resp, err := client.Repositories.ListCollaborators(*repo.Owner.Login, *repo.Name, opt)
	if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusForbidden {
		return nil
	}
	if err != nil {
		return err
	}

	keys, resp, err := client.Repositories.ListKeys(*repo.Owner.Login, *repo.Name, opt)
	if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusForbidden {
		return nil
	}
	if err != nil {
		return err
	}

	hooks, resp, err := client.Repositories.ListHooks(*repo.Owner.Login, *repo.Name, opt)
	if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusForbidden {
		return nil
	}
	if err != nil {
		return err
	}

	branches, _, err := client.Repositories.ListBranches(*repo.Owner.Login, *repo.Name, opt)
	if err != nil {
		return err
	}
	protectedBranches := []string{}
	for _, branch := range branches {
		if *branch.Protection.Enabled {
			protectedBranches = append(protectedBranches, *branch.Name)
		}
	}

	// only print whole status if we have more that one collaborator
	if len(collabs) <= 1 && len(keys) < 1 && len(hooks) < 1 && len(protectedBranches) < 1 {
		return nil
	}

	output := fmt.Sprintf("%s -> \n", *repo.FullName)

	if len(collabs) > 1 {
		logins := []string{}
		for _, c := range collabs {
			logins = append(logins, *c.Login)
		}
		output += fmt.Sprintf("\tCollaborators (%d): %s\n", len(logins), strings.Join(logins, ", "))
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
