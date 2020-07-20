module github.com/monasuite/monawallet/wallet/txauthor

go 1.12

require (
	github.com/monasuite/monad v0.22.1-beta
	github.com/monasuite/monautil v1.1.1
	github.com/monasuite/monawallet/wallet/txrules v1.1.0
	github.com/monasuite/monawallet/wallet/txsizes v1.1.0
)

replace github.com/monasuite/monawallet/wallet/txrules => ../txrules

replace github.com/monasuite/monawallet/wallet/txsizes => ../txsizes
