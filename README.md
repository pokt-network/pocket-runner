# pocket-runner
Pocket Runner is an task runner, that runs [pocket-core binary](https://github.com/pokt-network/pocket-core) as a subtask, listens events on planned upgrades and automatically updates pocket-core.

## How To Use
Before using you must have have a directory containing the following struct
```
/
  runner/
    genesis/
      bin/<name_of_binary>
    upgrades/
      <UPGRADE_NAME>/
        bin/<name_of_binary>
```

```
go build -o pocket-runner main.go
env DAEMON_HOME=<path_to_your_runne_dir> env DAEMON_NAME=<your_daemon_name> ./pocket-runner start --blockTime 1
```
Once Running using `SIGTERM` or `SIGINT` will cause a graceful shutdown.

The pocket-runner will run your `runner/genesis/bin` until an upgrade has been processed on the chain. It will then wait for the specific block height and the next release.

NOTE: pocket-runner will pass all arguments to the running binary, for a full list of valid arguments check the [pocket-core cli spec](https://github.com/pokt-network/pocket-core/blob/staging/doc/cli-interface-spec.md)


## Auto-Download
By passing in the env `DAEMON_ALLOW_DOWNLOAD="on"` you will enable auto download, which will verify wether the upgrade is available, if the upgrade is not available it will get a the source code of the release & build it. 

P.D: As of right now only one mirror exists our [GitHub Releases archive](https://github.com/pokt-network/pocket-core/releases/)

## Testing
In order to run tests use the default go tool
```
go test 
```
NOTE: The `custom-core` included in `/x/runner/testdata` is a pocket-core binary built for linux; running on any other OS will cause an error on the tests. to mitigate this you may replace `custom-core` with any build of [pocket-core](https://github.com/pokt-network/pocket-core) for your OS.

## Contributing
Please read [CONTRIBUTING.md](CONTRIBUTING.md) for details on contributions and the process of submitting pull requests.
