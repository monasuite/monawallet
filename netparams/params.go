// Copyright (c) 2013-2015 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package netparams

import "github.com/monasuite/monad/chaincfg"

// Params is used to group parameters for various networks such as the main
// network and test networks.
type Params struct {
	*chaincfg.Params
	RPCClientPort string
	RPCServerPort string
}

// MainNetParams contains parameters specific running monawallet and
// monad on the main network (wire.MainNet).
var MainNetParams = Params{
	Params:        &chaincfg.MainNetParams,
	RPCClientPort: "9400",
	RPCServerPort: "9402",
}

// TestNet4Params contains parameters specific running monawallet and
// monad on the test network (version 3) (wire.TestNet4).
var TestNet4Params = Params{
	Params:        &chaincfg.TestNet4Params,
	RPCClientPort: "19400",
	RPCServerPort: "19402",
}

// SimNetParams contains parameters specific to the simulation test network
// (wire.SimNet).
var SimNetParams = Params{
	Params:        &chaincfg.SimNetParams,
	RPCClientPort: "18556",
	RPCServerPort: "18554",
}
