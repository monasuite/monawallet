// Copyright (c) 2013-2016 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package txrules

import (
	"testing"

	"github.com/monasuite/monad/wire"
	"github.com/monasuite/monautil"
)

// TestDust tests the isDust API.
func TestDust(t *testing.T) {
	pkScript := []byte{0x76, 0xa9, 0x21, 0x03, 0x2f, 0x7e, 0x43,
		0x0a, 0xa4, 0xc9, 0xd1, 0x59, 0x43, 0x7e, 0x84, 0xb9,
		0x75, 0xdc, 0x76, 0xd9, 0x00, 0x3b, 0xf0, 0x92, 0x2c,
		0xf3, 0xaa, 0x45, 0x28, 0x46, 0x4b, 0xab, 0x78, 0x0d,
		0xba, 0x5e, 0x88, 0xac}

	tests := []struct {
		name     string // test description
		txOut    wire.TxOut
		relayFee monautil.Amount // minimum relay transaction fee.
		isDust   bool
	}{
		{
			// Any value is allowed with a zero relay fee.
			"zero value with zero relay fee",
			wire.TxOut{Value: 0, PkScript: pkScript},
			0,
			false,
		},
		//{
		//	// Zero value is dust with any relay fee"
		//	"zero value with very small tx fee",
		//	wire.TxOut{Value: 0, PkScript: pkScript},
		//	1,
		//	true,
		//},
		{
			"38 byte public key script with value 584",
			wire.TxOut{Value: 584, PkScript: pkScript},
			1000,
			true,
		},
		{
			"38 byte public key script with value 585",
			wire.TxOut{Value: 585, PkScript: pkScript},
			1000,
			false,
		},
		{
			// Maximum allowed value is never dust.
			"max satoshi amount is never dust",
			wire.TxOut{Value: monautil.MaxSatoshi, PkScript: pkScript},
			monautil.MaxSatoshi,
			false,
		},
		//{
		//	// Maximum int64 value causes overflow.
		//	"maximum int64 value",
		//	wire.TxOut{Value: 1<<63 - 1, PkScript: pkScript},
		//	1<<63 - 1,
		//	true,
		//},
		{
			// Unspendable pkScript due to an invalid public key
			// script.
			"unspendable pkScript",
			wire.TxOut{Value: 5000, PkScript: []byte{0x01}},
			0, // no relay fee
			true,
		},
	}
	for _, test := range tests {
		res := IsDustOutput(&test.txOut, test.relayFee)
		if res != test.isDust {
			t.Fatalf("Dust test '%s' failed: want %v got %v",
				test.name, test.isDust, res)
			continue
		}
	}
}
