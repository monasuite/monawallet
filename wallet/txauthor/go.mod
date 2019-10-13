module github.com/monasuite/monawallet/wallet/txauthor

go 1.12

require (
	github.com/btcsuite/btcd v0.0.0-20190824003749-130ea5bddde3
	github.com/btcsuite/btcutil v0.0.0-20190425235716-9e5f4b9a998d
	github.com/monasuite/monawallet/txrules v0.0.0-20190719070505-7e52842c5c05
	github.com/monasuite/monawallet/txsizes v0.0.0-20190719070505-7e52842c5c05
)

replace github.com/monasuite/monawallet/wallet/txrules => ../txrules

replace github.com/monasuite/monawallet/wallet/txsizes => ../txsizes
