module github.com/likecoin/likechain

go 1.14

require (
	github.com/cosmos/cosmos-sdk v0.40.0-rc0
	github.com/cosmos/gaia v0.0.1-0.20201013155758-3a8b1b414004
	github.com/gorilla/mux v1.8.0
	github.com/rakyll/statik v0.1.7
	github.com/spf13/cast v1.3.1
	github.com/spf13/cobra v1.0.0
	github.com/tendermint/tendermint v0.34.0-rc4.0.20201005135527-d7d0ffea13c6
	github.com/tendermint/tm-db v0.6.2
	go.etcd.io/etcd v0.0.0-20191023171146-3cf2f69b5738
)

replace github.com/gogo/protobuf => github.com/regen-network/protobuf v1.3.2-alpha.regen.4
