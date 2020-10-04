package cmds

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"math/big"
	"os"
	"testing"

	"github.com/spikeekips/mitum-currency/currency"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/encoder"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/stretchr/testify/suite"
)

type testDesign struct {
	suite.Suite
	Encs    *encoder.Encoders
	JSONEnc *jsonenc.Encoder
}

func (t *testDesign) SetupSuite() {
	t.Encs = encoder.NewEncoders()

	t.JSONEnc = jsonenc.NewEncoder()
	_ = t.Encs.AddEncoder(t.JSONEnc)

	_ = t.Encs.AddHinter(key.BTCPrivatekeyHinter)
	_ = t.Encs.AddHinter(key.BTCPublickeyHinter)
	_ = t.Encs.AddHinter(base.StringAddress(""))
}

func (t *testDesign) createCerts() (string, string) {
	priv, err := util.GenerateED25519Privatekey()
	t.NoError(err)

	template := x509.Certificate{SerialNumber: big.NewInt(1)}
	template.DNSNames = append(template.DNSNames, "localhost")

	certDER, err := x509.CreateCertificate(
		rand.Reader,
		&template,
		&template,
		priv.Public().(ed25519.PublicKey),
		priv,
	)
	t.NoError(err)

	keyBytes, err := x509.MarshalPKCS8PrivateKey(priv)
	t.NoError(err)

	keyFile, err := ioutil.TempFile("/tmp", "prefix")
	t.NoError(err)

	certFile, err := ioutil.TempFile("/tmp", "prefix")
	t.NoError(err)

	t.NoError(pem.Encode(
		keyFile,
		&pem.Block{
			Type:  "PRIVATE KEY",
			Bytes: keyBytes,
		},
	))
	t.NoError(pem.Encode(
		certFile,
		&pem.Block{
			Type:  "CERTIFICATE",
			Bytes: certDER,
		},
	))

	return keyFile.Name(), certFile.Name()
}

func (t *testDesign) TestNew() {
	y := `
address: mc-node-010a:0.0.1
privatekey: KxaTHDAQnmFeWWik5MqWXBYkhvp5EpWbsZzXeHDdTDb5NE1dVw8w-0112:0.0.1
storage: mongodb://127.0.0.1:27017/mc
blockfs: ./blockfs
network-id: mc; Thu 10 Sep 2020 03:23:31 PM UTC
network:
    bind: quic://0.0.0.0:54321
    publish: quic://127.0.0.1:54321
component:
    fee-amount:
        type: ratio
        min: 20
        ratio: 0.2
`

	d, err := LoadNodeDesign([]byte(y), t.Encs)
	t.NoError(err)

	address, err := base.NewStringAddress("mc-node")
	t.NoError(err)
	t.True(address.Equal(d.Address()))

	priv, err := key.NewBTCPrivatekeyFromString("KxaTHDAQnmFeWWik5MqWXBYkhvp5EpWbsZzXeHDdTDb5NE1dVw8w")
	t.NoError(err)
	t.True(priv.Equal(d.Privatekey()))

	t.Equal([]byte("mc; Thu 10 Sep 2020 03:23:31 PM UTC"), d.NetworkID())
	t.Equal("mongodb://127.0.0.1:27017/mc", d.Storage)
	t.Equal("quic://0.0.0.0:54321", d.Network.Bind().String())
	t.Equal("quic://127.0.0.1:54321", d.Network.Publish)

	t.NotEmpty(d.FeeAmount)
	t.IsType(currency.RatioFeeAmount{}, d.FeeAmount)
	t.Equal(`{"type": "ratio", "ratio": 0.200000, "min": "20"}`, d.FeeAmount.Verbose())

	t.NotNil(d.Digest)
}

func (t *testDesign) TestDigestWithoutCertificates() {
	y := `
address: mc-node-010a:0.0.1
privatekey: KxaTHDAQnmFeWWik5MqWXBYkhvp5EpWbsZzXeHDdTDb5NE1dVw8w-0112:0.0.1
storage: mongodb://127.0.0.1:27017/mc
blockfs: ./blockfs
network-id: mc; Thu 10 Sep 2020 03:23:31 PM UTC
network:
    bind: quic://0.0.0.0:54321
    publish: quic://127.0.0.1:54321
component:
    fee-amount:
        type: ratio
        min: 20
        ratio: 0.2
    digest:
        storage: mongodb://127.0.0.1:27017/mc-digest
        cache: memory://
        network:
            bind: https://0.0.0.0:8090
            publish: https://showme:4430
`

	_, err := LoadNodeDesign([]byte(y), t.Encs)
	t.Contains(err.Error(), "missing certificates for https")
}

func (t *testDesign) TestDigest() {
	y := `
address: mc-node-010a:0.0.1
privatekey: KxaTHDAQnmFeWWik5MqWXBYkhvp5EpWbsZzXeHDdTDb5NE1dVw8w-0112:0.0.1
storage: mongodb://127.0.0.1:27017/mc
blockfs: ./blockfs
network-id: mc; Thu 10 Sep 2020 03:23:31 PM UTC
network:
    bind: quic://0.0.0.0:54321
    publish: quic://127.0.0.1:54321
component:
    fee-amount:
        type: ratio
        min: 20
        ratio: 0.2
    digest:
        storage: mongodb://127.0.0.1:27017/mc-digest
        cache: memory://
        network:
            bind: https://0.0.0.0:8090
            publish: https://showme:4430
            cert-key: "%s"
            cert: "%s"
`

	keyFile, certFile := t.createCerts()
	defer os.Remove(keyFile)
	defer os.Remove(certFile)

	d, err := LoadNodeDesign([]byte(fmt.Sprintf(y, keyFile, certFile)), t.Encs)
	t.NoError(err)

	address, err := base.NewStringAddress("mc-node")
	t.NoError(err)
	t.True(address.Equal(d.Address()))

	priv, err := key.NewBTCPrivatekeyFromString("KxaTHDAQnmFeWWik5MqWXBYkhvp5EpWbsZzXeHDdTDb5NE1dVw8w")
	t.NoError(err)
	t.True(priv.Equal(d.Privatekey()))

	t.Equal([]byte("mc; Thu 10 Sep 2020 03:23:31 PM UTC"), d.NetworkID())
	t.Equal("mongodb://127.0.0.1:27017/mc", d.Storage)
	t.Equal("quic://0.0.0.0:54321", d.Network.Bind().String())
	t.Equal("quic://127.0.0.1:54321", d.Network.Publish)

	t.NotEmpty(d.FeeAmount)
	t.IsType(currency.RatioFeeAmount{}, d.FeeAmount)
	t.Equal(`{"type": "ratio", "ratio": 0.200000, "min": "20"}`, d.FeeAmount.Verbose())

	t.Equal("https://0.0.0.0:8090", d.Digest.Network.Bind().String())
	t.Equal("https://showme:4430", d.Digest.Network.Publish)
	t.Equal("mongodb://127.0.0.1:27017/mc-digest", d.Digest.Storage)
	t.Equal("memory://", d.Digest.Cache)
}

func (t *testDesign) TestEmptyDigest() {
	y := `
address: mc-node-010a:0.0.1
privatekey: KxaTHDAQnmFeWWik5MqWXBYkhvp5EpWbsZzXeHDdTDb5NE1dVw8w-0112:0.0.1
storage: mongodb://127.0.0.1:27017/mc
blockfs: ./blockfs
network-id: mc; Thu 10 Sep 2020 03:23:31 PM UTC
network:
    bind: quic://0.0.0.0:54321
    publish: quic://127.0.0.1:54321
`

	d, err := LoadNodeDesign([]byte(y), t.Encs)
	t.NoError(err)

	t.Nil(d.Digest.Network)
	t.Equal("mongodb://127.0.0.1:27017/mc", d.Digest.Storage)
	t.Equal(DefaultDigestCacheStrign, d.Digest.Cache)
}

func (t *testDesign) TestEmptyDigestMissingPublish() {
	y := `
address: mc-node-010a:0.0.1
privatekey: KxaTHDAQnmFeWWik5MqWXBYkhvp5EpWbsZzXeHDdTDb5NE1dVw8w-0112:0.0.1
storage: mongodb://127.0.0.1:27017/mc
blockfs: ./blockfs
network-id: mc; Thu 10 Sep 2020 03:23:31 PM UTC
network:
    bind: quic://0.0.0.0:54321
    publish: quic://127.0.0.1:54321
component:
    digest:
        network:
            bind: https://0.0.0.0:4430
            cert-key: "%s"
            cert: "%s"
`

	keyFile, certFile := t.createCerts()
	defer os.Remove(keyFile)
	defer os.Remove(certFile)

	d, err := LoadNodeDesign([]byte(fmt.Sprintf(y, keyFile, certFile)), t.Encs)
	t.NoError(err)

	t.Equal(fmt.Sprintf("https://0.0.0.0:%d", DefaultDigestPort), d.Digest.Network.Bind().String())
	t.Equal(fmt.Sprintf("https://127.0.0.1:%d", DefaultDigestPort), d.Digest.Network.Publish)
	t.Equal("mongodb://127.0.0.1:27017/mc", d.Digest.Storage)
	t.Equal(DefaultDigestCacheStrign, d.Digest.Cache)
}

func TestDesign(t *testing.T) {
	suite.Run(t, new(testDesign))
}
