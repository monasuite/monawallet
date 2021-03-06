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
	github.com/monasuite/monad v0.22.1-beta.0.20200923041908-5682eaeb0bcc
	github.com/monasuite/monautil v1.1.3-0.20201206184148-63353a3ceadd
	github.com/monasuite/monautil/psbt v1.1.0
	github.com/monasuite/monawallet/wallet/txauthor v1.1.0
	github.com/monasuite/monawallet/wallet/txrules v1.1.0
	github.com/monasuite/monawallet/walletdb v1.3.4
	github.com/monasuite/monawallet/wtxmgr v1.2.0
	github.com/shopspring/decimal v1.2.0
	github.com/stretchr/testify v1.5.1
	golang.org/x/crypto v0.0.0-20200820211705-5c72a883971a
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
