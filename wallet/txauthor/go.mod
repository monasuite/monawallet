module github.com/monasuite/monawallet/wallet/txauthor

go 1.12

require (
	github.com/monasuite/monad v0.20.1-beta
	github.com/monasuite/monautil v0.0.0-20190606162653-90b266792864
	github.com/monasuite/monawallet/wallet/txrules v1.0.0
	github.com/monasuite/monawallet/wallet/txsizes v1.0.0
)

replace github.com/monasuite/monawallet/wallet/txrules => ../txrules

replace github.com/monasuite/monawallet/wallet/txsizes => ../txsizes
