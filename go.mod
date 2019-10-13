module github.com/monasuite/monawallet

require (
<<<<<<< HEAD
	github.com/btcsuite/btclog v0.0.0-20170628155309-84c8d2346e9f
	github.com/btcsuite/golangcrypto v0.0.0-20150304025918-53f62d9b43e8
=======
	github.com/btcsuite/btcd v0.0.0-20190824003749-130ea5bddde3
	github.com/btcsuite/btclog v0.0.0-20170628155309-84c8d2346e9f
	github.com/btcsuite/btcutil v0.0.0-20190425235716-9e5f4b9a998d
	github.com/btcsuite/btcwallet/wallet/txauthor v1.0.0
	github.com/btcsuite/btcwallet/wallet/txrules v1.0.0
	github.com/btcsuite/btcwallet/walletdb v1.0.0
	github.com/btcsuite/btcwallet/wtxmgr v1.0.0
>>>>>>> upstream/master
	github.com/btcsuite/websocket v0.0.0-20150119174127-31079b680792
	github.com/davecgh/go-spew v1.1.1
	github.com/golang/protobuf v1.3.1
	github.com/jessevdk/go-flags v1.4.0
	github.com/jrick/logrotate v1.0.0
<<<<<<< HEAD
	github.com/lightninglabs/gozmq v0.0.0-20190710231225-cea2a031735d
	github.com/monasuite/monad v0.0.0-20190614102927-b024b3975103
	github.com/monasuite/monautil v0.0.0-20190606162653-90b266792864
	github.com/monasuite/neutrino v0.0.0-20190606165554-f2e38dac24d6
	go.etcd.io/bbolt v1.3.3 // indirect
	golang.org/x/crypto v0.0.0-20190605123033-f99c8df09eb5
	golang.org/x/net v0.0.0-20190603091049-60506f45cf65
=======
	github.com/kkdai/bstream v0.0.0-20181106074824-b3251f7901ec // indirect
	github.com/lightninglabs/gozmq v0.0.0-20190710231225-cea2a031735d
	github.com/lightninglabs/neutrino v0.0.0-20190906012717-f087198de655
	golang.org/x/crypto v0.0.0-20190211182817-74369b46fc67
	golang.org/x/net v0.0.0-20190206173232-65e2d4e15006
	google.golang.org/genproto v0.0.0-20190201180003-4b09977fb922 // indirect
>>>>>>> upstream/master
	google.golang.org/grpc v1.18.0
)

replace github.com/btcsuite/btcwallet/walletdb => ./walletdb

replace github.com/btcsuite/btcwallet/wtxmgr => ./wtxmgr

replace github.com/btcsuite/btcwallet/wallet/txauthor => ./wallet/txauthor

replace github.com/btcsuite/btcwallet/wallet/txrules => ./wallet/txrules

replace github.com/btcsuite/btcwallet/wallet/txsizes => ./wallet/txsizes

go 1.13
