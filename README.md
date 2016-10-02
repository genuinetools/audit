# audit

[![Travis CI](https://travis-ci.org/jessfraz/audit.svg?branch=master)](https://travis-ci.org/jessfraz/audit)

For checking what collaborators, hooks, deploy keys, and protected branched
you have added on all your GitHub repositories. This also scans all an
organizations repos you have permission to view.
Because nobody has enough RAM in their brain to remember this stuff for 100+ repos.

## Usage

```console
$ audit -h
audit - v0.1.0
  -d    run in debug mode
  -token string
        GitHub API token
  -v    print version and exit (shorthand)
  -version
        print version and exit
```

```console
$ audit --token 12345
jessfraz/apk-file ->
        Hooks (1):
                travis - active:true (https://api.github.com/repos/jessfraz/apk-file/hooks/8426605)
        Protected Branches (1): master
--

jessfraz/apparmor-docs ->
        Keys (1):
                jenkins - ro:false (https://api.github.com/repos/jessfraz/apparmor-docs/keys/18549738)
--

jessfraz/bane ->
        Hooks (1):
                travis - active:true (https://api.github.com/repos/jessfraz/bane/hooks/6178025)
        Protected Branches (1): master
--

jessfraz/battery ->
        Hooks (1):
                travis - active:true (https://api.github.com/repos/jessfraz/battery/hooks/8388640)
        Protected Branches (1): master
--

jessfraz/irssi ->
	Collaborators (3): tianon, jessfraz, docker-library-bot
	Hooks (1):
		docker - active:true (https://api.github.com/repos/jessfraz/irssi/hooks/3918042)
	Protected Branches (1): master
--
```
