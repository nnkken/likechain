module github.com/likecoin/likechain

go 1.14

require (
	github.com/cosmos/cosmos-sdk v0.40.0-rc2
	github.com/gorilla/mux v1.8.0
	github.com/pkg/errors v0.9.1
	github.com/rakyll/statik v0.1.7
	github.com/spf13/cast v1.3.1
	github.com/spf13/cobra v1.1.1
	github.com/tendermint/go-amino v0.16.0
	github.com/tendermint/tendermint v0.34.0-rc5
	github.com/tendermint/tm-db v0.6.2
	go.etcd.io/etcd v0.0.0-20191023171146-3cf2f69b5738
)

replace github.com/gogo/protobuf => github.com/regen-network/protobuf v1.3.2-alpha.regen.4
