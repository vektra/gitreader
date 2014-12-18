gitreader
=========

There are times, in a developers life, when they need to read data out of a git repository.

In those times, there are 3 paths to take:

1. Shell out to the git subcommands to pull the data out
1. Use an API that just shells out to git subcommands
1. Use an API that implements parts of git itself

For various reasons, there are times where #1 and #2 are clunky and unacceptable.
For those times, we have APIs that implement parts of git. This is one of those APIs.

This specific API implements a golang library that contains only the functionality
to read a git repository on disk. No remote protocols, no ability to write new data,
only reading.

So when you're in golang and need to read some data out a git repository, reach for
**gitreader**, you'll be happy with yourself.

 - Vektra Devs
