# Developer tools for version software artifacts

This repository contains several tools for handling meta data and artifacts produced by software development activities.

The general idea is to produce both libraries for development artifact handling and also tools that can be invoked to perform administrative functions related to software release and build activities.

With the advent of the new github.com/karlmutch/duat (developer utilities and tools) project the dump-ver tool is being replaced with facilicities offered within the duat project.

## bump-ver
A version bumping tool for storing and manipuating the semantic version within files that contain a HTML formatted version tag.

This tool also has the ability to generate version string using git branch names when the pre command is used.

This tool applies the Semantic Versioning 2.0 spec to version strings in files, https://semver.org/.  When the pre command is used the version bumping will append to the version string an increasing pre-release string that can be used to sort the versions creating precedence for versions when they are used with containers or other assets.

Version : <repo-version>0.1.0</repo-version>

# Installation

This go can installed using the following command:

```shell
$ go get -u karlmutch/bump-ver
```

# Basic usage

bump-ver by default will read your README.md file and will examine it for HTML markup embeeded in the file `&lt;repo-version&gt;[semver]&lt;/repo-version&gt;`.  The text within the tag will be parsed and validated as being valid semver, if this fails the command will exit.  Once parsed the options specified on the bump-ver command line will be used to morph the version and written back into the file.

bump-ver can also be used with the apply option to modify files based upon the version within an authorative file.  When this option is used not changes are made to the existing input file.  This command is only for propagating an existing version to other files.

bump-ver will output to stdout the new version number, except for the apply command where you will get the current version applies to the target-file list.

The command has the following usage:

<doc-opt><code>
Bump Version Tag

Usage:

  bump-ver [major | major | minor | pre | extract | apply] [-f=<input-file>] [-t=[<target-file>,...]]

Options:
  -h --help              Show this message.
  -version               Show the version of this software.
  -f=<input-file>        A file containing an HTML repo-version tag to be morped or queried [default: README.md]
  -t=<target-file>,...   A comma seperated list of files that will be examined for version tags and modified based upon the input-file version
</code></doc-opt>
