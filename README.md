# torev
Tor reverse proxy, for transparently exposing your websites (or any TCP program) as Tor hidden services

## Installation
Step 1: [Install Go](https://golang.org/)

Step 2: Run `go get -u -v github.com/360ied/torev` (Note that this also updates the program.)

Note: As this uses [go-libtor](github.com/ipsn/go-libtor), this does not require Tor to be installed, as it is statically linked into the program!

## Usage
Run `torev`. It will generate a configuration file. You can edit it if you wish, but the defaults should be fine for most web services.

Once it has started, it will print the address of the hidden service into stdout. Look out for the `[STARTED]` message and the really long and random string. That's what you want to put in your browser (or other program).
