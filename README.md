# ghb0t

[![Travis CI](https://travis-ci.org/jfrazelle/ghb0t.svg?branch=master)](https://travis-ci.org/jfrazelle/ghb0t)

A GitHub Bot to automatically delete your fork's branches after a pull request
has been merged.

> **NOTE:** This will **never** delete a branch named "master" AND will
**never** delete a branch that is not owned by the current authenticated user.
If the pull request is closed _without_ merging, it will **not** delete it.

## Usage

```
$ ghb0t -h
ghb0t - v0.1.0
  -d    run in debug mode
  -seconds int
        seconds to wait before checking for new events (default 30)
  -token string
        GitHub API token
  -v    print version and exit (shorthand)
  -version
        print version and exit
```
