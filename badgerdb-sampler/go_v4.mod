module github.com/oasisprotocol/badgerdb-sampler

go 1.25.4

require (
	github.com/dgraph-io/badger/v4 v4.2.0
	github.com/gogo/protobuf v1.3.2
	github.com/tendermint/tendermint v0.34.21
)

// Note: Dependencies will be filled in when v4 support is implemented
// Run 'go mod tidy' with -modfile=go.mod.v4 to populate
