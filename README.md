# audit

[![Travis CI](https://travis-ci.org/genuinetools/audit.svg?branch=master)](https://travis-ci.org/genuinetools/audit)

For checking what collaborators, hooks, deploy keys, and protected branched
you have added on all your GitHub repositories. This also scans all an
organizations repos you have permission to view.
Because nobody has enough RAM in their brain to remember this stuff for 100+ repos.

Check out [genuinetools/pepper](https://github.com/genuinetools/pepper) for setting all your GitHub repos master branches
to be protected. Even has settings for organizations and a dry-run flag for the paranoid.

## Installation

#### Binaries

- **darwin** [386](https://github.com/genuinetools/audit/releases/download/v0.4.1/audit-darwin-386) / [amd64](https://github.com/genuinetools/audit/releases/download/v0.4.1/audit-darwin-amd64)
- **freebsd** [386](https://github.com/genuinetools/audit/releases/download/v0.4.1/audit-freebsd-386) / [amd64](https://github.com/genuinetools/audit/releases/download/v0.4.1/audit-freebsd-amd64)
- **linux** [386](https://github.com/genuinetools/audit/releases/download/v0.4.1/audit-linux-386) / [amd64](https://github.com/genuinetools/audit/releases/download/v0.4.1/audit-linux-amd64) / [arm](https://github.com/genuinetools/audit/releases/download/v0.4.1/audit-linux-arm) / [arm64](https://github.com/genuinetools/audit/releases/download/v0.4.1/audit-linux-arm64)
- **solaris** [amd64](https://github.com/genuinetools/audit/releases/download/v0.4.1/audit-solaris-amd64)
- **windows** [386](https://github.com/genuinetools/audit/releases/download/v0.4.1/audit-windows-386) / [amd64](https://github.com/genuinetools/audit/releases/download/v0.4.1/audit-windows-amd64)

#### Via Go

```bash
$ go get github.com/genuinetools/audit
```

## Usage

```console
$ audit -h
                 _ _ _
  __ _ _   _  __| (_) |_
 / _` | | | |/ _` | | __|
| (_| | |_| | (_| | | |_
 \__,_|\__,_|\__,_|_|\__|

 Auditing what collaborators, hooks, and deploy keys you have added on all your GitHub repositories.
 Version: v0.4.1
 Build: a55701b

  -d    run in debug mode
  -owner
        only audit repos the token owner owns
  -repo string
        specific repo to test (e.g. 'genuinetools/audit')
  -token string
        GitHub API token (or env var GITHUB_TOKEN)
  -v    print version and exit (shorthand)
  -version
        print version and exit
```

```console
$ audit --token 12345
genuinetools/apk-file ->
        Hooks (1):
                travis - active:true (https://api.github.com/repos/genuinetools/apk-file/hooks/8426605)
        Protected Branches (1): master
--

genuinetools/apparmor-docs ->
        Keys (1):
                jenkins - ro:false (https://api.github.com/repos/genuinetools/apparmor-docs/keys/18549738)
        Unprotected Branches (1): master
--

genuinetools/bane ->
        Hooks (1):
                travis - active:true (https://api.github.com/repos/genuinetools/bane/hooks/6178025)
        Protected Branches (1): master
--

genuinetools/battery ->
        Hooks (1):
                travis - active:true (https://api.github.com/repos/genuinetools/battery/hooks/8388640)
        Protected Branches (1): master
        Unprotected Branches (1): WIP
--

genuinetools/irssi ->
	Collaborators (3): tianon, genuinetools, docker-library-bot
	Hooks (1):
		docker - active:true (https://api.github.com/repos/genuinetools/irssi/hooks/3918042)
	Protected Branches (1): master
--
```
