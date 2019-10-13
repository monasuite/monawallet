module github.com/monasuite/monawallet

require (
	github.com/btcsuite/btclog v0.0.0-20170628155309-84c8d2346e9f
	github.com/btcsuite/btcwallet/walletdb v1.0.0
	github.com/btcsuite/websocket v0.0.0-20150119174127-31079b680792
	github.com/davecgh/go-spew v1.1.1
	github.com/golang/protobuf v1.3.1
	github.com/jessevdk/go-flags v1.4.0
	github.com/jrick/logrotate v1.0.0
	github.com/lightninglabs/gozmq v0.0.0-20190710231225-cea2a031735d
	github.com/monasuite/monad v0.0.0-20190614102927-b024b3975103
	github.com/monasuite/monautil v0.0.0-20190606162653-90b266792864
	github.com/monasuite/neutrino v0.0.0-20190606165554-f2e38dac24d6
	go.etcd.io/bbolt v1.3.3 // indirect
	golang.org/x/crypto v0.0.0-20190211182817-74369b46fc67
	golang.org/x/net v0.0.0-20190206173232-65e2d4e15006
	google.golang.org/genproto v0.0.0-20190201180003-4b09977fb922 // indirect
	google.golang.org/grpc v1.18.0
)

replace github.com/btcsuite/btcwallet/walletdb => ./walletdb

replace github.com/monasuite/monawallet/wtxmgr => ./wtxmgr

replace github.com/monasuite/monawallet/wallet/txauthor => ./wallet/txauthor

replace github.com/monasuite/monawallet/wallet/txrules => ./wallet/txrules

replace github.com/monasuite/monawallet/wallet/txsizes => ./wallet/txsizes

go 1.13
