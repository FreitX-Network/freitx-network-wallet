# freitx-network-wallet

Wallet Service for [FreitX Network blockchain](https://github.com/freitx-project/freitx-network-blockchain).

## Minimum requirements

| Components | Version | Description |
|----------|-------------|-------------|
|[Golang](https://golang.org) | >= 1.10.2 | The Go Programming Language |

### Setup Dev Environment
```
mkdir -p ~/go/src/github.com/freitx-project
cd ~/go/src/github.com/freitx-project
git clone git@github.com:freitx-project/freitx-network-wallet.git
cd freitx-network-wallet
```

Install Go dependency management tool from [golang dep](https://github.com/golang/dep) first and then

```dep ensure --vendor-only```

```make fmt; make build```

Note: If your Dev Environment is in Ubuntu, you need to export the following Path:

LD_LIBRARY_PATH=$LD_LIBRARY_PATH:$GOPATH/src/github.com/freitx-project/freitx-network-wallet/vendor/github.com/freitx-project/freitx-network-blockchain/crypto/lib:$GOPATH/src/github.com/freitx-project/freitx-network-wallet/vendor/github.com/freitx-project/freitx-network-blockchain/crypto/lib/blslib

### Run unit tests
```make test```

### Run wallet server with default configurations
```make run``` to start wallet server

### Run wallet server with customized configurations
`./bin/server`
