module github.com/monasuite/monawallet

require (
	github.com/btcsuite/btclog v0.0.0-20170628155309-84c8d2346e9f
	github.com/btcsuite/websocket v0.0.0-20150119174127-31079b680792
	github.com/davecgh/go-spew v1.1.1
	github.com/golang/protobuf v1.3.1
	github.com/jessevdk/go-flags v1.4.0
	github.com/jrick/logrotate v1.0.0
	github.com/lightninglabs/gozmq v0.0.0-20191113021534-d20a764486bf
	github.com/monaarchives/neutrino v0.11.1
	github.com/monasuite/monad v0.20.1-beta
	github.com/monasuite/monautil v0.0.0-20190606162653-90b266792864
	github.com/monasuite/monawallet/wallet/txauthor v1.0.0
	github.com/monasuite/monawallet/wallet/txrules v1.0.0
	github.com/monasuite/monawallet/wallet/txsizes v1.0.0
	github.com/monasuite/monawallet/walletdb v1.1.0
	github.com/monasuite/monawallet/wtxmgr v1.0.0
	golang.org/x/crypto v0.0.0-20190424203555-c05e17bb3b2d
	golang.org/x/net v0.0.0-20190424112056-4829fb13d2c6
	google.golang.org/genproto v0.0.0-20190201180003-4b09977fb922 // indirect
	google.golang.org/grpc v1.18.0
)

replace github.com/monasuite/monawallet/walletdb => ./walletdb

replace github.com/monasuite/monawallet/wtxmgr => ./wtxmgr

replace github.com/monasuite/monawallet/wallet/txauthor => ./wallet/txauthor

replace github.com/monasuite/monawallet/wallet/txrules => ./wallet/txrules

replace github.com/monasuite/monawallet/wallet/txsizes => ./wallet/txsizes

go 1.13
