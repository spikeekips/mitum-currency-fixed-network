### mitum-currency

[![CircleCI](https://img.shields.io/circleci/project/github/spikeekips/mitum-currency/proto3.svg?style=flat-square&logo=circleci&label=circleci&cacheSeconds=60)](https://circleci.com/gh/spikeekips/mitum-currency/tree/proto3)
[![Documentation](https://readthedocs.org/projects/mitum-currency-doc/badge/?version=proto3)](https://mitum-currency-doc.readthedocs.io/en/latest/?badge=master)
[![GoDoc](https://godoc.org/github.com/golang/gddo?status.svg)](https://pkg.go.dev/github.com/spikeekips/mitum-currency?tab=overview)
[![Go Report Card](https://goreportcard.com/badge/github.com/spikeekips/mitum-currency)](https://goreportcard.com/report/github.com/spikeekips/mitum-currency)
[![codecov](https://codecov.io/gh/spikeekips/mitum-currency/branch/proto3/graph/badge.svg)](https://codecov.io/gh/spikeekips/mitum-currency)
[![](https://tokei.rs/b1/github/spikeekips/mitum-currency?category=lines)](https://github.com/spikeekips/mitum-currency)

*mitum-currency* is the cryptocurrency case of mitum model, based on
[*mitum*](https://github.com/spikeekips/mitum). This project was started for
creating the first model case of *mitum*, but it can be used for simple
cryptocurrency blockchain network (at your own risk).

~~For more details, see the [documentation](https://mitum-currency-doc.readthedocs.io/en/latest/?badge=master).~~

#### Features,

* account: account address and keypair is not same.
* simple transaction: creating account, transfer balance.
* supports multiple keypairs: *btc*, *ethereum*, *stellar* keypairs.
* *mongodb*: as mitum does, *mongodb* is the primary storage.

#### Installation

```sh
$ go get -u github.com/spikeekips/mitum-currency/cmd/mc
```

Or, build it from source
```sh
$ cd mitum-currency
$ go build -o mc ./cmd/mc/main.go
```

#### Run

At the first time, you can simply start node with example configuration.

> To start, you need to run *mongodb* on localhost(port, 27017).

```
$ cd mitum-currency
$ mc init cmd/mc/standalone.yml
$ mc run  cmd/mc/standalone.yml
```

> Please check `$ mc --help` for detailed usage.

#### Test

```sh
$ go clean -testcache; time go test -race -tags 'test' -v -timeout 1m ./... -run .
```
