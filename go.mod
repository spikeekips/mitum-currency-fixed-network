module github.com/spikeekips/mitum-currency-fixed-network

go 1.16

require (
	github.com/alecthomas/kong v0.2.20
	github.com/bluele/gcache v0.0.2
	github.com/btcsuite/btcutil v1.0.3-0.20201208143702-a53e38424cce
	github.com/gorilla/handlers v1.5.1
	github.com/gorilla/mux v1.8.0
	github.com/json-iterator/go v1.1.12
	github.com/pkg/errors v0.9.1
	github.com/rainycape/memcache v0.0.0-20150622160815-1031fa0ce2f2
	github.com/rs/zerolog v1.26.0
	github.com/spikeekips/mitum v0.0.0-20211228033330-da1863767169
	github.com/spikeekips/mitum-currency v0.0.0-00010101000000-000000000000
	github.com/stretchr/testify v1.7.0
	github.com/ulule/limiter/v3 v3.9.0
	go.mongodb.org/mongo-driver v1.8.0
	golang.org/x/net v0.0.0-20211206223403-eba003a116a9
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b
)

replace github.com/spikeekips/mitum-currency => ./

replace github.com/spikeekips/mitum => github.com/spikeekips/mitum-fixed-network v0.0.0-20230719202330-5a6e01ea13b3
