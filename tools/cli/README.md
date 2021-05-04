Documentation for the Cadence command line interface is located at our [main site](https://cadenceworkflow.io/docs/cli/).

## Quick Start
Run `make cadence` from the project root. You should see an executable file called `cadence`. Try a few example commands to 
get started:   
`./cadence` for help on top level commands and global options   
`./cadence domain` for help on domain operations  
`./cadence workflow` for help on workflow operations  
`./cadence tasklist` for help on tasklist operations  
(`./cadence help`, `./cadence help [domain|workflow]` will also print help messages)

**Note:** Make sure you have a Cadence server running before using the CLI.

## Homebrew
Cadence CLI homebrew formula is maintain by [a community project](https://github.com/git-hulk/homebrew-cadence): 
```
% brew tap git-hulk/cadence
% brew install cadence-cli
```
