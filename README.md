# Mesos Bitcoin Miner

A Mesos framework that mines Bitcoin.

## Build

You need to have [Go](http://golang.org/) installed.  Then in this directory:

    go get github.com/tools/godep # only if you don't have godep installed already
    godep go build ./...

Now you should have a binary called `scheduler` inside the directory.

## Usage

    ./scheduler -master="<address of mesos master>" -bitcoindAddress="<address of your bitcoin daemon>" <your bitcoin RPC username> <your bitcoin RPC password>

    Note that you can usually find your bitcoin RPC username and password in your `bitcoin.conf` file.

    There is a demo here: http://recordit.co/4KOIUuLuqu

## Disclaimer

With the rise of GPUs and ASICs, mining Bitcoin using raw CPUs has been unprofitable for the past few years.  Therefore, this project is probably only useful for time travellers from 2011 or before.  Although now that I think about it, there is no Mesos before 2011, so you probably still can't use it even if you are a time traveller.  Sorry guys.
