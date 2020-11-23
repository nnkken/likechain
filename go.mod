module github.com/likecoin/likechain

go 1.14

require (
	github.com/cosmos/cosmos-sdk v0.40.0
	github.com/gorilla/mux v1.8.0
	github.com/ipfs/go-cid v0.0.7
	github.com/ipfs/go-ipld-cbor v0.0.5
	github.com/likecoin/iscn-ipld v0.0.0-20200517153629-3078d7917930
	github.com/multiformats/go-multibase v0.0.3
	github.com/pkg/errors v0.9.1
	github.com/polydawn/refmt v0.0.0-20201211092308-30ac6d18308e
	github.com/rakyll/statik v0.1.7
	github.com/spf13/cast v1.3.1
	github.com/spf13/cobra v1.1.1
	github.com/spf13/viper v1.7.1
	github.com/tendermint/go-amino v0.16.0
	github.com/tendermint/tendermint v0.34.1
	github.com/tendermint/tm-db v0.6.3
	go.etcd.io/etcd v0.0.0-20191023171146-3cf2f69b5738
)

replace github.com/gogo/protobuf => github.com/regen-network/protobuf v1.3.2-alpha.regen.4
