# audit

[![Travis CI](https://travis-ci.org/jessfraz/audit.svg?branch=master)](https://travis-ci.org/jessfraz/audit)

For checking what collaborators, hooks, and deploy keys you have added on all
your GitHub repositories. Because nobody has enough RAM in their brain to
remember this stuff for 100+ repos.

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
--

jessfraz/apparmor-docs ->
        Keys (1):
                jenkins - ro:false (https://api.github.com/repos/jessfraz/apparmor-docs/keys/18549738)
--

jessfraz/bane ->
        Hooks (1):
                travis - active:true (https://api.github.com/repos/jessfraz/bane/hooks/6178025)
--

jessfraz/battery ->
        Hooks (1):
                travis - active:true (https://api.github.com/repos/jessfraz/battery/hooks/8388640)
--
```
