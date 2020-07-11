module github.com/spikeekips/mitum-currency

go 1.14

replace github.com/spikeekips/mitum => /workspace/mitum/src

require (
	github.com/alecthomas/kong v0.2.11
	github.com/btcsuite/btcutil v1.0.2
	github.com/klauspost/compress v1.10.10 // indirect
	github.com/marten-seemann/qtls v0.10.0 // indirect
	github.com/spikeekips/mitum v0.0.0-20200711094947-02dc72d9d883
	github.com/stretchr/testify v1.6.1
	go.mongodb.org/mongo-driver v1.3.5
	go.uber.org/automaxprocs v1.3.0
	golang.org/x/sync v0.0.0-20200625203802-6e8e738ad208 // indirect
	golang.org/x/sys v0.0.0-20200625212154-ddb9806d33ae // indirect
	golang.org/x/xerrors v0.0.0-20191204190536-9bdfabe68543
	google.golang.org/protobuf v1.25.0 // indirect
	gopkg.in/yaml.v3 v3.0.0-20200615113413-eeeca48fe776
)
