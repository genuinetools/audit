# audit

[![Travis CI](https://img.shields.io/travis/genuinetools/audit.svg?style=for-the-badge)](https://travis-ci.org/genuinetools/audit)
[![GoDoc](https://img.shields.io/badge/godoc-reference-5272B4.svg?style=for-the-badge)](https://godoc.org/github.com/genuinetools/audit)
[![Github All Releases](https://img.shields.io/github/downloads/genuinetools/audit/total.svg?style=for-the-badge)](https://github.com/genuinetools/audit/releases)

For checking what collaborators, hooks, deploy keys, and protected branches
you have added on all your GitHub repositories. This also scans all an
organization's repos you have permission to view.
Because nobody has enough RAM in their brain to remember this stuff for 100+ repos.

Check out [genuinetools/pepper](https://github.com/genuinetools/pepper) for setting all your GitHub repo's master branches
to be protected. Pepper even has settings for organizations and a dry-run flag for the paranoid.

**Table of Contents**

<!-- toc -->

<!-- tocstop -->

## Installation

#### Binaries

For installation instructions from binaries please visit the [Releases Page](https://github.com/genuinetools/audit/releases).

#### Via Go

```console
$ go get github.com/genuinetools/audit
```

## Usage

```console
$ audit -h
audit -  Tool to audit what collaborators, hooks, and deploy keys are on your GitHub repositories.

Usage: audit <command>

Flags:

  -d      enable debug logging (default: false)
  -owner  only audit repos the token owner owns (default: false)
  -orgs   specific orgs to check (e.g. 'genuinetools')
  -repo   specific repo to test (e.g. 'genuinetools/audit') (default: <none>)
  -token  GitHub API token (or env var GITHUB_TOKEN)

Commands:

  version  Show the version information.
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
