# Mesos Bitcoin Miner

A Mesos framework that mines Bitcoin.

## Setup

To run this framework, you need to have a Bitcoin daemon (`bitcoind`) and a Mesos cluster running.

### Bitcoind

There are many ways to install `bitcoind`; you should be able to easily find instructions online for your platform.  For OS X, I usually just [build it from source](https://github.com/bitcoin/bitcoin/blob/master/doc/build-osx.md).

In order for the Mesos cluster to talk to your `bitcoind`, you will need to configure your `bitcoind` to enable RPC.  The location of the bitcoind config file is platform-specific, but for OS X it's at:

    /Users/${USER}/Library/Application Support/Bitcoin/bitcoin.conf

If the file doesn't exist, go ahead and create it.  You will need to set three fields: `rpcuser`, `rpcpassword`, and `rpcallowip`.  Here is how my config file looks like:

    rpcuser=bitcoinrpc
    rpcpassword=BEBSruNHg14Tmz4D6X8qCjA22RCuoniDxovRCvPvGqS1
    rpcallowip=0.0.0.0/0

`rpcuser` and `rpcpassword` can be arbitrary strings.

Note that for the purpose of the demo, I've set `rpcallowip` to the entire IP space.  This is **highly insecure**.  In practice, you will most definitely want to set it to only allow for some specific IPs.

Once the config file is in place, run bitcoind in server mode:

    bitcoind --daemon

`bitcoind` will now start downloading the blockchain.  The framework will not work until the entire blockchain has been downloaded.  Note that the blockchain is huge, so you might need to wait a while.  To check how many blocks you have downloaded so far:

    bitcoin-cli getblockcount

### Mesos Cluster

To set up a real Mesos cluster, refer to the [official documentation](http://mesos.apache.org/documentation/latest/).

To set up a local cluster using Vagrant, [this](https://github.com/everpeace/vagrant-mesos) and [this](https://github.com/mesosphere/playa-mesos) work really well.

Importantly, you need to make sure that you start your Mesos slaves with the Docker containerizer enabled, since we make use of Docker images in our framework.  If you start the Mesos slaves manually, you can just set the environment variable `MESOS_CONTAINERIZERS` to `docker`.  Otherwise, you might need to modify the config files under `/etc/mesos`.

In the end, you should obtain the IP address and port where the Mesos master is running on.  They are needed to run the framework.

## Build

You need to have [Go](http://golang.org/) installed.  Then in this directory:

    go get github.com/tools/godep # only if you don't have godep installed already
    godep go build ./...

Now you should have a binary called `scheduler` inside the directory.

## Usage

    ./scheduler -master="<address of mesos master>" -bitcoindAddress="<address of your bitcoin daemon>" <your bitcoin RPC username> <your bitcoin RPC password>

For example, in my local setup, I run:

    ./scheduler -master="172.31.1.11:5050" -bitcoindAddress="172.31.2.1" bitcoinrpc BEBSruNHg14Tmz4D6X8qCjA22RCuoniDxovRCvPvGqS1

Note that the `bitcoindAddress` is the IP address of your `bitcoind` **from the perspective of the Mesos cluster**.

## Demo

http://recordit.co/4KOIUuLuqu

## Disclaimer

With the rise of GPUs and ASICs, mining Bitcoin using raw CPUs has been unprofitable for the past few years.  Therefore, this project is probably only useful for time travellers from 2011 or before.  Although now that I think about it, Mesos doesn't even exist before 2011, so even time travellers won't be able to use it.  Sorry guys.

## License

[Apache 2.0](http://www.apache.org/licenses/LICENSE-2.0)
