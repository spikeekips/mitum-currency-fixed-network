### mitum-currency

[![CircleCI](https://img.shields.io/circleci/project/github/spikeekips/mitum-currency/master.svg?style=flat-square&logo=circleci&label=circleci&cacheSeconds=60)](https://circleci.com/gh/spikeekips/mitum-currency/tree/master)
[![Documentation](https://readthedocs.org/projects/mitum-currency-doc/badge/?version=master)](https://mitum-currency-doc.readthedocs.io/en/latest/?badge=master)
[![GoDoc](https://godoc.org/github.com/golang/gddo?status.svg)](https://pkg.go.dev/github.com/spikeekips/mitum-currency?tab=overview)
[![Go Report Card](https://goreportcard.com/badge/github.com/spikeekips/mitum-currency)](https://goreportcard.com/report/github.com/spikeekips/mitum-currency)
[![codecov](https://codecov.io/gh/spikeekips/mitum-currency/branch/master/graph/badge.svg)](https://codecov.io/gh/spikeekips/mitum-currency)
[![](https://tokei.rs/b1/github/spikeekips/mitum-currency?category=lines)](https://github.com/spikeekips/mitum-currency)
[![Total alerts](https://img.shields.io/lgtm/alerts/g/spikeekips/mitum-currency.svg?logo=lgtm&logoWidth=18)](https://lgtm.com/projects/g/spikeekips/mitum-currency/alerts/)

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
* supports multiple currencies

#### Installation

> NOTE: at this time, *mitum* and *mitum-currency* is actively developed, so
before building mitum-currency, you will be better with building the latest
mitum source.
> `$ git clone https://github.com/spikeekips/mitum`
>
> and then, add `replace github.com/spikeekips/mitum => <your mitum source directory>` to `go.mod` of *mitum-currency*.

Build it from source
```sh
$ cd mitum-currency
$ go build -ldflags="-X 'main.Version=v0.0.1'" -o ./mc ./main.go
```

#### Run

At the first time, you can simply start node with example configuration.

> To start, you need to run *mongodb* on localhost(port, 27017).

```
$ ./mc node init ./standalone.yml
$ ./mc node run ./standalone.yml
```

> Please check `$ ./mc --help` for detailed usage.

#### Test

```sh
$ go clean -testcache; time go test -race -tags 'test' -v -timeout 20m ./... -run .
```
