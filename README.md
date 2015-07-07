# Mesos Bitcoin Miner

A Mesos framework that mines Bitcoin.

## Build

You need to have [Go](http://golang.org/) installed.  Then in this directory:

    go get github.com/tools/godep # only if you don't have godep installed already
    godep go build ./...

Now you should have a binary called `scheduler` inside the directory.

## Usage

    ./scheduler -master="<address of mesos master>" -bitcoindAddress="<address of your bitcoin daemon>" <your bitcoin RPC username> <your bitcoin RPC password>

    Note that you can usually find your bitcoin RPC username and password in your `bitcoin.conf` file.  You might also need to set `rpcallowip` to the IP range of your mesos cluster so they can access your bitcoind.

    Also, since this framework makes use of docker images, you will need to start mesos slaves with the docker containerizer.  This can be done by setting the environment variable `MESOS_CONTAINERIZERS` to `docker`.

## Demo

http://recordit.co/4KOIUuLuqu

## Disclaimer

With the rise of GPUs and ASICs, mining Bitcoin using raw CPUs has been unprofitable for the past few years.  Therefore, this project is probably only useful for time travellers from 2011 or before.  Although now that I think about it, Mesos doesn't even exist before 2011, so even time travellers won't be able to use it.  Sorry guys.
