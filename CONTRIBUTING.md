# Developing Cadence

This doc is intended for contributors to `cadence` server (hopefully that's you!)

>Note: All contributors also need to fill out the [Uber Contributor License Agreement](http://t.uber.com/cla) before we can merge in any of your changes

## Development Environment

* Golang. Install on OS X with `brew install go`. 
>Note: If running into any compiling issue, make sure you upgrade to the latest stable version of Golang. 

## Checking out the code

Make sure the repository is cloned to the correct location:

```bash
cd $GOPATH
git clone https://github.com/uber/cadence.git src/github.com/uber/cadence
cd $GOPATH/src/github.com/uber/cadence
```

## Licence headers

This project is Open Source Software, and requires a header at the beginning of
all source files. To verify that all files contain the header execute:

```bash
make copyright
```

## Commit Messages And Titles of Pull Requests

Overcommit adds some requirements to your commit messages. At Uber, we follow the
[Chris Beams](http://chris.beams.io/posts/git-commit/) guide to writing git
commit messages. Read it, follow it, learn it, love it.

All commit messages are from the titles of your pull requests. So make sure follow the rules when titling them. 
Please don't use very generic titles like "bug fixes". 

All PR titles should start with UPPER case.

Examples:

- [Make sync activity retry multiple times before fetch history from remote](https://github.com/uber/cadence/pull/1379)
- [Enable archival config per domain](https://github.com/uber/cadence/pull/1351)

## Issues to start with

Take a look at the list of issues labeled 
[up-for-grabs](https://github.com/uber/cadence/labels/up-for-grabs). These issues 
are a great way to start contributing to Cadence.

## Building

You can compile the `cadence` service and helper tools without running test:

```bash
make bins
```

## Testing

>Note: The default setup of Cadence depends on Cassandra and Kafka(is being deprecated). 
This section assumes you are testing with them, too. Please refer to [persistence documentation](https://github.com/uber/cadence/blob/master/docs/persistence.md) if you want to test with others like MySQL/Postgres. 

Before running the tests you must have `cassandra`, `kafka`, and its `zookeeper` dependency:

```bash
# install cassandra
# you can reduce memory used by cassandra (by default 4GB), by following instructions here: http://codefoundries.com/developer/cassandra/cassandra-installation-mac.html
brew install cassandra

# install zookeeper compatible with jdk@8 that OSX has
brew install https://raw.githubusercontent.com/Homebrew/homebrew-core/6d8197bbb5f77e62d51041a3ae552ce2f8ff1344/Formula/zookeeper.rb

# install kafka compatible with zookeeper 3.4.14
brew install --ignore-dependencies https://raw.githubusercontent.com/Homebrew/homebrew-core/6d8197bbb5f77e62d51041a3ae552ce2f8ff1344/Formula/kafka.rb

# start services
brew services start cassandra
brew services start zookeeper
brew services start kafka

```

Run all the tests:

```bash
make test

# `make test` currently do not include crossdc tests, start kafka and run 
make test_xdc

# or go to folder with *_test.go, e.g
cd service/history/ 
go test -v
# run single test
go test -v <path> -run <TestSuite> -testify.m <TestSpercificTaskName>
# example:
go test -v github.com/uber/cadence/common/persistence/persistence-tests -run TestVisibilitySamplingSuite -testify.m TestListClosedWorkflowExecutions
```
