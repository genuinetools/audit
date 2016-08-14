# audit

[![Travis CI](https://travis-ci.org/jfrazelle/audit.svg?branch=master)](https://travis-ci.org/jfrazelle/audit)

For checking what collaborators, hooks, and deploy keys you have added on all
your GitHub repositorys. Because nobody has enough RAM in their brain to
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
jfrazelle/apk-file ->
        Hooks (1):
                travis - active:true (https://api.github.com/repos/jfrazelle/apk-file/hooks/8426605)
--

jfrazelle/apparmor-docs ->
        Keys (1):
                jenkins - ro:false (https://api.github.com/repos/jfrazelle/apparmor-docs/keys/18549738)
--

jfrazelle/bane ->
        Hooks (1):
                travis - active:true (https://api.github.com/repos/jfrazelle/bane/hooks/6178025)
--

jfrazelle/battery ->
        Hooks (1):
                travis - active:true (https://api.github.com/repos/jfrazelle/battery/hooks/8388640)
--
```
