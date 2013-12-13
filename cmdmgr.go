/*
 * Copyright (c) 2013 Conformal Systems LLC <info@conformal.com>
 *
 * Permission to use, copy, modify, and distribute this software for any
 * purpose with or without fee is hereby granted, provided that the above
 * copyright notice and this permission notice appear in all copies.
 *
 * THE SOFTWARE IS PROVIDED "AS IS" AND THE AUTHOR DISCLAIMS ALL WARRANTIES
 * WITH REGARD TO THIS SOFTWARE INCLUDING ALL IMPLIED WARRANTIES OF
 * MERCHANTABILITY AND FITNESS. IN NO EVENT SHALL THE AUTHOR BE LIABLE FOR
 * ANY SPECIAL, DIRECT, INDIRECT, OR CONSEQUENTIAL DAMAGES OR ANY DAMAGES
 * WHATSOEVER RESULTING FROM LOSS OF USE, DATA OR PROFITS, WHETHER IN AN
 * ACTION OF CONTRACT, NEGLIGENCE OR OTHER TORTIOUS ACTION, ARISING OUT OF
 * OR IN CONNECTION WITH THE USE OR PERFORMANCE OF THIS SOFTWARE.
 */

package main

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/conformal/btcjson"
	"github.com/conformal/btcutil"
	"github.com/conformal/btcwallet/tx"
	"github.com/conformal/btcwallet/wallet"
	"github.com/conformal/btcwire"
	"github.com/conformal/btcws"
	"time"
)

var (
	// ErrBtcdDisconnected describes an error where an operation cannot
	// successfully complete due to btcd not being connected to
	// btcwallet.
	ErrBtcdDisconnected = errors.New("btcd disconnected")
)

type cmdHandler func(chan []byte, btcjson.Cmd)

var rpcHandlers = map[string]cmdHandler{
	// Standard bitcoind methods
	"dumpprivkey":           DumpPrivKey,
	"getaddressesbyaccount": GetAddressesByAccount,
	"getbalance":            GetBalance,
	"getnewaddress":         GetNewAddress,
	"importprivkey":         ImportPrivKey,
	"listaccounts":          ListAccounts,
	"listtransactions":      ListTransactions,
	"sendfrom":              SendFrom,
	"sendmany":              SendMany,
	"settxfee":              SetTxFee,
	"walletlock":            WalletLock,
	"walletpassphrase":      WalletPassphrase,

	// Extensions not exclusive to websocket connections.
	"createencryptedwallet": CreateEncryptedWallet,
}

// Extensions exclusive to websocket connections.
var wsHandlers = map[string]cmdHandler{
	"getaddressbalance":   GetAddressBalance,
	"getbalances":         GetBalances,
	"listalltransactions": ListAllTransactions,
	"walletislocked":      WalletIsLocked,
}

// ProcessRequest checks the requests sent from a frontend.  If the
// request method is one that must be handled by btcwallet, the
// request is processed here.  Otherwise, the request is sent to btcd
// and btcd's reply is routed back to the frontend.
func ProcessRequest(frontend chan []byte, msg []byte, ws bool) {
	// Parse marshaled command and check
	cmd, err := btcjson.ParseMarshaledCmd(msg)
	if err != nil {
		// Check that msg is valid JSON-RPC.  Reply to frontend
		// with error if invalid.
		if cmd == nil {
			ReplyError(frontend, nil, &btcjson.ErrInvalidRequest)
			return
		}

		// btcwallet cannot handle this command, so defer handling
		// to btcd.
		DeferToBTCD(frontend, msg)
		return
	}

	// Check for a handler to reply to cmd.  If none exist, defer to btcd.
	if f, ok := rpcHandlers[cmd.Method()]; ok {
		f(frontend, cmd)
	} else if f, ok := wsHandlers[cmd.Method()]; ws && ok {
		f(frontend, cmd)
	} else {
		// btcwallet does not have a handler for the command.  Pass
		// to btcd and route replies back to the appropiate frontend.
		DeferToBTCD(frontend, msg)
	}
}

// DeferToBTCD sends an unmarshaled command to btcd, modifying the id
// and setting up a reply route to route the reply from btcd back to
// the frontend reply channel with the original id.
func DeferToBTCD(frontend chan []byte, msg []byte) {
	// msg cannot be sent to btcd directly, but the ID must instead be
	// changed to include additonal routing information so replies can
	// be routed back to the correct frontend.  Unmarshal msg into a
	// generic btcjson.Message struct so the ID can be modified and the
	// whole thing re-marshaled.
	var m btcjson.Message
	json.Unmarshal(msg, &m)

	// Create a new ID so replies can be routed correctly.
	n := <-NewJSONID
	var id interface{} = RouteID(m.Id, n)
	m.Id = &id

	// Marshal the request with modified ID.
	newMsg, err := json.Marshal(m)
	if err != nil {
		log.Errorf("DeferToBTCD: Cannot marshal message: %v", err)
		return
	}

	// If marshaling suceeded, save the id and frontend reply channel
	// so the reply can be sent to the correct frontend.
	replyRouter.Lock()
	replyRouter.m[n] = frontend
	replyRouter.Unlock()

	// Send message with modified ID to btcd.
	btcdMsgs <- newMsg
}

// RouteID creates a JSON-RPC id for a frontend request that was deferred
// to btcd.
func RouteID(origID, routeID interface{}) string {
	return fmt.Sprintf("btcwallet(%v)-%v", routeID, origID)
}

// ReplyError creates and marshals a btcjson.Reply with the error e,
// sending the reply to a frontend reply channel.
func ReplyError(frontend chan []byte, id interface{}, e *btcjson.Error) {
	// Create a Reply with a non-nil error to marshal.
	r := btcjson.Reply{
		Error: e,
		Id:    &id,
	}

	// Marshal reply and send to frontend if marshaling suceeded.
	if mr, err := json.Marshal(r); err == nil {
		frontend <- mr
	}
}

// ReplySuccess creates and marshals a btcjson.Reply with the result r,
// sending the reply to a frontend reply channel.
func ReplySuccess(frontend chan []byte, id interface{}, result interface{}) {
	// Create a Reply with a non-nil result to marshal.
	r := btcjson.Reply{
		Result: result,
		Id:     &id,
	}

	// Marshal reply and send to frontend if marshaling suceeded.
	if mr, err := json.Marshal(r); err == nil {
		frontend <- mr
	}
}

// DumpPrivKey replies to a dumpprivkey request with the private
// key for a single address, or an appropiate error if the wallet
// is locked.
func DumpPrivKey(frontend chan []byte, icmd btcjson.Cmd) {
	// Type assert icmd to access parameters.
	cmd, ok := icmd.(*btcjson.DumpPrivKeyCmd)
	if !ok {
		ReplyError(frontend, icmd.Id(), &btcjson.ErrInternal)
		return
	}

	switch key, err := accountstore.DumpWIFPrivateKey(cmd.Address); err {
	case nil:
		// Key was found.
		ReplySuccess(frontend, cmd.Id(), key)

	case wallet.ErrWalletLocked:
		// Address was found, but the private key isn't
		// accessible.
		ReplyError(frontend, cmd.Id(), &btcjson.ErrWalletUnlockNeeded)

	default: // all other non-nil errors
		e := &btcjson.Error{
			Code:    btcjson.ErrWallet.Code,
			Message: err.Error(),
		}
		ReplyError(frontend, cmd.Id(), e)
	}
}

// DumpWallet replies to a dumpwallet request with all private keys
// in a wallet, or an appropiate error if the wallet is locked.
// TODO: finish this to match bitcoind by writing the dump to a file.
func DumpWallet(frontend chan []byte, icmd btcjson.Cmd) {
	// Type assert icmd to access parameters.
	cmd, ok := icmd.(*btcjson.DumpWalletCmd)
	if !ok {
		ReplyError(frontend, icmd.Id(), &btcjson.ErrInternal)
		return
	}

	switch keys, err := accountstore.DumpKeys(); err {
	case nil:
		// Reply with sorted WIF encoded private keys
		ReplySuccess(frontend, cmd.Id(), keys)

	case wallet.ErrWalletLocked:
		ReplyError(frontend, cmd.Id(), &btcjson.ErrWalletUnlockNeeded)

	default: // any other non-nil error
		e := &btcjson.Error{
			Code:    btcjson.ErrWallet.Code,
			Message: err.Error(),
		}
		ReplyError(frontend, cmd.Id(), e)
		return
	}
}

// GetAddressesByAccount replies to a getaddressesbyaccount request with
// all addresses for an account, or an error if the requested account does
// not exist.
func GetAddressesByAccount(frontend chan []byte, icmd btcjson.Cmd) {
	// Type assert icmd to access parameters.
	cmd, ok := icmd.(*btcjson.GetAddressesByAccountCmd)
	if !ok {
		ReplyError(frontend, icmd.Id(), &btcjson.ErrInternal)
		return
	}

	switch a, err := accountstore.Account(cmd.Account); err {
	case nil:
		// Reply with sorted active payment addresses.
		ReplySuccess(frontend, cmd.Id(), a.SortedActivePaymentAddresses())

	case ErrAcctNotExist:
		ReplyError(frontend, cmd.Id(),
			&btcjson.ErrWalletInvalidAccountName)

	default: // all other non-nil errors
		e := &btcjson.Error{
			Code:    btcjson.ErrWallet.Code,
			Message: err.Error(),
		}
		ReplyError(frontend, cmd.Id(), e)
	}
}

// GetBalance replies to a getbalance request with the balance for an
// account (wallet), or an error if the requested account does not
// exist.
func GetBalance(frontend chan []byte, icmd btcjson.Cmd) {
	// Type assert icmd to access parameters.
	cmd, ok := icmd.(*btcjson.GetBalanceCmd)
	if !ok {
		ReplyError(frontend, icmd.Id(), &btcjson.ErrInternal)
		return
	}

	balance, err := accountstore.CalculateBalance(cmd.Account, cmd.MinConf)
	if err != nil {
		ReplyError(frontend, cmd.Id(),
			&btcjson.ErrWalletInvalidAccountName)
		return
	}

	// Reply with calculated balance.
	ReplySuccess(frontend, cmd.Id(), balance)
}

// GetBalances replies to a getbalances extension request by notifying
// the frontend of all balances for each opened account.
func GetBalances(frontend chan []byte, cmd btcjson.Cmd) {
	NotifyBalances(frontend)
}

// GetAddressBalance replies to a getaddressbalance extension request
// by replying with the current balance (sum of unspent transaction
// output amounts) for a single address.
func GetAddressBalance(frontend chan []byte, icmd btcjson.Cmd) {
	// Type assert icmd to access parameters.
	cmd, ok := icmd.(*btcws.GetAddressBalanceCmd)
	if !ok {
		ReplyError(frontend, icmd.Id(), &btcjson.ErrInternal)
		return
	}

	// Is address valid?
	pkhash, net, err := btcutil.DecodeAddress(cmd.Address)
	if err != nil || net != cfg.Net() {
		ReplyError(frontend, cmd.Id(), &btcjson.ErrInvalidAddressOrKey)
		return
	}

	// Look up account which holds this address.
	aname, err := LookupAccountByAddress(cmd.Address)
	if err == ErrNotFound {
		e := &btcjson.Error{
			Code:    btcjson.ErrInvalidAddressOrKey.Code,
			Message: "Address not found in wallet",
		}
		ReplyError(frontend, cmd.Id(), e)
		return
	}

	// Get the account which holds the address in the request.
	// This should not fail, so if it does, return an internal
	// error to the frontend.
	a, err := accountstore.Account(aname)
	if err != nil {
		ReplyError(frontend, cmd.Id(), &btcjson.ErrInternal)
		return
	}

	bal := a.CalculateAddressBalance(pkhash, int(cmd.Minconf))
	ReplySuccess(frontend, cmd.Id(), bal)
}

// ImportPrivKey replies to an importprivkey request by parsing
// a WIF-encoded private key and adding it to an account.
func ImportPrivKey(frontend chan []byte, icmd btcjson.Cmd) {
	// Type assert icmd to access parameters.
	cmd, ok := icmd.(*btcjson.ImportPrivKeyCmd)
	if !ok {
		ReplyError(frontend, icmd.Id(), &btcjson.ErrInternal)
		return
	}

	// Get the acount included in the request. Yes, Label is the
	// account name...
	a, err := accountstore.Account(cmd.Label)
	switch err {
	case nil:
		break

	case ErrAcctNotExist:
		ReplyError(frontend, cmd.Id(),
			&btcjson.ErrWalletInvalidAccountName)
		return

	default:
		e := &btcjson.Error{
			Code:    btcjson.ErrWallet.Code,
			Message: err.Error(),
		}
		ReplyError(frontend, cmd.Id(), e)
		return
	}

	// Import the private key, handling any errors.
	switch err := a.ImportPrivKey(cmd.PrivKey, cmd.Rescan); err {
	case nil:
		// If the import was successful, reply with nil.
		ReplySuccess(frontend, cmd.Id(), nil)

	case wallet.ErrWalletLocked:
		ReplyError(frontend, cmd.Id(), &btcjson.ErrWalletUnlockNeeded)

	default:
		e := &btcjson.Error{
			Code:    btcjson.ErrWallet.Code,
			Message: err.Error(),
		}
		ReplyError(frontend, cmd.Id(), e)
	}
}

// NotifyBalances notifies an attached frontend of the current confirmed
// and unconfirmed account balances.
//
// TODO(jrick): Switch this to return a single JSON object
// (map[string]interface{}) of all accounts and their balances, instead of
// separate notifications for each account.
func NotifyBalances(frontend chan []byte) {
	accountstore.NotifyBalances(frontend)
}

// GetNewAddress responds to a getnewaddress request by getting a new
// address for an account.  If the account does not exist, an appropiate
// error is returned to the frontend.
func GetNewAddress(frontend chan []byte, icmd btcjson.Cmd) {
	// Type assert icmd to access parameters.
	cmd, ok := icmd.(*btcjson.GetNewAddressCmd)
	if !ok {
		ReplyError(frontend, icmd.Id(), &btcjson.ErrInternal)
		return
	}

	a, err := accountstore.Account(cmd.Account)
	switch err {
	case nil:
		break

	case ErrAcctNotExist:
		ReplyError(frontend, cmd.Id(),
			&btcjson.ErrWalletInvalidAccountName)
		return

	case ErrBtcdDisconnected:
		e := &btcjson.Error{
			Code:    btcjson.ErrInternal.Code,
			Message: "btcd disconnected",
		}
		ReplyError(frontend, cmd.Id(), e)
		return

	default: // all other non-nil errors
		e := &btcjson.Error{
			Code:    btcjson.ErrWallet.Code,
			Message: err.Error(),
		}
		ReplyError(frontend, cmd.Id(), e)
		return
	}

	addr, err := a.NewAddress()
	switch err {
	case nil:
		// Reply with the new payment address string.
		ReplySuccess(frontend, cmd.Id(), addr)

	case wallet.ErrWalletLocked:
		// The wallet is locked error may be sent if the keypool needs
		// to be refilled, but the wallet is currently in a locked
		// state.  Notify the frontend that an unlock is needed to
		// refill the keypool.
		ReplyError(frontend, cmd.Id(), &btcjson.ErrWalletKeypoolRanOut)

	default: // all other non-nil errors
		e := &btcjson.Error{
			Code:    btcjson.ErrWallet.Code,
			Message: err.Error(),
		}
		ReplyError(frontend, cmd.Id(), e)
	}
}

// ListAccounts replies to a listaccounts request by returning a JSON
// object mapping account names with their balances.
func ListAccounts(frontend chan []byte, icmd btcjson.Cmd) {
	// Type assert icmd to access parameters.
	cmd, ok := icmd.(*btcjson.ListAccountsCmd)
	if !ok {
		ReplyError(frontend, icmd.Id(), &btcjson.ErrInternal)
		return
	}

	pairs := accountstore.ListAccounts(cmd.MinConf)

	// Reply with the map.  This will be marshaled into a JSON object.
	ReplySuccess(frontend, cmd.Id(), pairs)
}

// ListTransactions replies to a listtransactions request by returning an
// array of JSON objects with details of sent and recevied wallet
// transactions.
func ListTransactions(frontend chan []byte, icmd btcjson.Cmd) {
	// Type assert icmd to access parameters.
	cmd, ok := icmd.(*btcjson.ListTransactionsCmd)
	if !ok {
		ReplyError(frontend, icmd.Id(), &btcjson.ErrInternal)
		return
	}

	a, err := accountstore.Account(cmd.Account)
	switch err {
	case nil:
		break

	case ErrAcctNotExist:
		ReplyError(frontend, cmd.Id(),
			&btcjson.ErrWalletInvalidAccountName)
		return

	default: // all other non-nil errors
		e := &btcjson.Error{
			Code:    btcjson.ErrWallet.Code,
			Message: err.Error(),
		}
		ReplyError(frontend, cmd.Id(), e)
		return
	}

	switch txList, err := a.ListTransactions(cmd.From, cmd.Count); err {
	case nil:
		// Reply with the list of tx information.
		ReplySuccess(frontend, cmd.Id(), txList)

	case ErrBtcdDisconnected:
		e := &btcjson.Error{
			Code:    btcjson.ErrInternal.Code,
			Message: "btcd disconnected",
		}
		ReplyError(frontend, cmd.Id(), e)

	default:
		e := &btcjson.Error{
			Code:    btcjson.ErrWallet.Code,
			Message: err.Error(),
		}
		ReplyError(frontend, cmd.Id(), e)
	}
}

// ListAllTransactions replies to a listtransactions request by returning
// an array of JSON objects with details of sent and recevied wallet
// transactions.  This is similar to ListTransactions, except it takes
// only a single optional argument for the account name and replies with
// all transactions.
func ListAllTransactions(frontend chan []byte, icmd btcjson.Cmd) {
	// Type assert icmd to access parameters.
	cmd, ok := icmd.(*btcws.ListAllTransactionsCmd)
	if !ok {
		ReplyError(frontend, icmd.Id(), &btcjson.ErrInternal)
		return
	}

	a, err := accountstore.Account(cmd.Account)
	switch err {
	case nil:
		break

	case ErrAcctNotExist:
		ReplyError(frontend, cmd.Id(),
			&btcjson.ErrWalletInvalidAccountName)
		return

	default: // all other non-nil errors
		e := &btcjson.Error{
			Code:    btcjson.ErrWallet.Code,
			Message: err.Error(),
		}
		ReplyError(frontend, cmd.Id(), e)
		return
	}

	switch txList, err := a.ListAllTransactions(); err {
	case nil:
		// Reply with the list of tx information.
		ReplySuccess(frontend, cmd.Id(), txList)

	case ErrBtcdDisconnected:
		e := &btcjson.Error{
			Code:    btcjson.ErrInternal.Code,
			Message: "btcd disconnected",
		}
		ReplyError(frontend, cmd.Id(), e)

	default:
		e := &btcjson.Error{
			Code:    btcjson.ErrWallet.Code,
			Message: err.Error(),
		}
		ReplyError(frontend, cmd.Id(), e)
	}
}

// SendFrom creates a new transaction spending unspent transaction
// outputs for a wallet to another payment address.  Leftover inputs
// not sent to the payment address or a fee for the miner are sent
// back to a new address in the wallet.  Upon success, the TxID
// for the created transaction is sent to the frontend.
func SendFrom(frontend chan []byte, icmd btcjson.Cmd) {
	// Type assert icmd to access parameters.
	cmd, ok := icmd.(*btcjson.SendFromCmd)
	if !ok {
		ReplyError(frontend, icmd.Id(), &btcjson.ErrInternal)
		return
	}

	// Check that signed integer parameters are positive.
	if cmd.Amount < 0 {
		e := &btcjson.Error{
			Code:    btcjson.ErrInvalidParameter.Code,
			Message: "amount must be positive",
		}
		ReplyError(frontend, cmd.Id(), e)
		return
	}
	if cmd.MinConf < 0 {
		e := &btcjson.Error{
			Code:    btcjson.ErrInvalidParameter.Code,
			Message: "minconf must be positive",
		}
		ReplyError(frontend, cmd.Id(), e)
		return
	}

	// Check that the account specified in the request exists.
	a, err := accountstore.Account(cmd.FromAccount)
	if err != nil {
		ReplyError(frontend, cmd.Id(),
			&btcjson.ErrWalletInvalidAccountName)
		return
	}

	// Create map of address and amount pairs.
	pairs := map[string]int64{
		cmd.ToAddress: cmd.Amount,
	}

	// Create transaction, replying with an error if the creation
	// was not successful.
	createdTx, err := a.txToPairs(pairs, cmd.MinConf)
	switch {
	case err == ErrNonPositiveAmount:
		e := &btcjson.Error{
			Code:    btcjson.ErrInvalidParameter.Code,
			Message: "amount must be positive",
		}
		ReplyError(frontend, cmd.Id(), e)
		return

	case err == wallet.ErrWalletLocked:
		ReplyError(frontend, cmd.Id(), &btcjson.ErrWalletUnlockNeeded)
		return

	case err != nil:
		e := &btcjson.Error{
			Code:    btcjson.ErrInternal.Code,
			Message: err.Error(),
		}
		ReplyError(frontend, cmd.Id(), e)
		return
	}

	// If a change address was added, mark wallet as dirty, sync to disk,
	// and Request updates for change address.
	if len(createdTx.changeAddr) != 0 {
		a.dirty = true
		if err := a.writeDirtyToDisk(); err != nil {
			log.Errorf("cannot write dirty wallet: %v", err)
		}
		a.ReqNewTxsForAddress(createdTx.changeAddr)
	}

	// Create sendrawtransaction request with hexstring of the raw tx.
	n := <-NewJSONID
	var id interface{} = fmt.Sprintf("btcwallet(%v)", n)
	m, err := btcjson.CreateMessageWithId("sendrawtransaction", id,
		hex.EncodeToString(createdTx.rawTx))
	if err != nil {
		e := &btcjson.Error{
			Code:    btcjson.ErrInternal.Code,
			Message: err.Error(),
		}
		ReplyError(frontend, cmd.Id(), e)
		return
	}

	// Set up a reply handler to respond to the btcd reply.
	replyHandlers.Lock()
	replyHandlers.m[n] = func(result interface{}, err *btcjson.Error) bool {
		return handleSendRawTxReply(frontend, cmd, result, err, a,
			createdTx)
	}
	replyHandlers.Unlock()

	// Send sendrawtransaction request to btcd.
	btcdMsgs <- m
}

// SendMany creates a new transaction spending unspent transaction
// outputs for a wallet to any number of  payment addresses.  Leftover
// inputs not sent to the payment address or a fee for the miner are
// sent back to a new address in the wallet.  Upon success, the TxID
// for the created transaction is sent to the frontend.
func SendMany(frontend chan []byte, icmd btcjson.Cmd) {
	// Type assert icmd to access parameters.
	cmd, ok := icmd.(*btcjson.SendManyCmd)
	if !ok {
		ReplyError(frontend, icmd.Id(), &btcjson.ErrInternal)
		return
	}

	// Check that minconf is positive.
	if cmd.MinConf < 0 {
		e := &btcjson.Error{
			Code:    btcjson.ErrInvalidParameter.Code,
			Message: "minconf must be positive",
		}
		ReplyError(frontend, cmd.Id(), e)
		return
	}

	// Check that the account specified in the request exists.
	a, err := accountstore.Account(cmd.FromAccount)
	if err != nil {
		ReplyError(frontend, cmd.Id(),
			&btcjson.ErrWalletInvalidAccountName)
		return
	}

	// Create transaction, replying with an error if the creation
	// was not successful.
	createdTx, err := a.txToPairs(cmd.Amounts, cmd.MinConf)
	switch {
	case err == ErrNonPositiveAmount:
		e := &btcjson.Error{
			Code:    btcjson.ErrInvalidParameter.Code,
			Message: "amount must be positive",
		}
		ReplyError(frontend, cmd.Id(), e)
		return

	case err == wallet.ErrWalletLocked:
		ReplyError(frontend, cmd.Id(), &btcjson.ErrWalletUnlockNeeded)
		return

	case err != nil:
		e := &btcjson.Error{
			Code:    btcjson.ErrInternal.Code,
			Message: err.Error(),
		}
		ReplyError(frontend, cmd.Id(), e)
		return
	}

	// If a change address was added, mark wallet as dirty, sync to disk,
	// and request updates for change address.
	if len(createdTx.changeAddr) != 0 {
		a.dirty = true
		if err := a.writeDirtyToDisk(); err != nil {
			log.Errorf("cannot write dirty wallet: %v", err)
		}
		a.ReqNewTxsForAddress(createdTx.changeAddr)
	}

	// Create sendrawtransaction request with hexstring of the raw tx.
	n := <-NewJSONID
	var id interface{} = fmt.Sprintf("btcwallet(%v)", n)
	m, err := btcjson.CreateMessageWithId("sendrawtransaction", id,
		hex.EncodeToString(createdTx.rawTx))
	if err != nil {
		e := &btcjson.Error{
			Code:    btcjson.ErrInternal.Code,
			Message: err.Error(),
		}
		ReplyError(frontend, cmd.Id(), e)
		return
	}

	// Set up a reply handler to respond to the btcd reply.
	replyHandlers.Lock()
	replyHandlers.m[n] = func(result interface{}, err *btcjson.Error) bool {
		return handleSendRawTxReply(frontend, cmd, result, err, a,
			createdTx)
	}
	replyHandlers.Unlock()

	// Send sendrawtransaction request to btcd.
	btcdMsgs <- m
}

func handleSendRawTxReply(frontend chan []byte, icmd btcjson.Cmd,
	result interface{}, e *btcjson.Error, a *Account,
	txInfo *CreatedTx) bool {

	if e != nil {
		log.Errorf("Could not send tx: %v", e.Message)
		ReplyError(frontend, icmd.Id(), e)
		return true
	}

	txIDStr, ok := result.(string)
	if !ok {
		e := &btcjson.Error{
			Code:    btcjson.ErrInternal.Code,
			Message: "Unexpected type from btcd reply",
		}
		ReplyError(frontend, icmd.Id(), e)
		return true
	}
	txID, err := btcwire.NewShaHashFromStr(txIDStr)
	if err != nil {
		e := &btcjson.Error{
			Code:    btcjson.ErrInternal.Code,
			Message: "Invalid hash string from btcd reply",
		}
		ReplyError(frontend, icmd.Id(), e)
		return true
	}

	// Add to transaction store.
	sendtx := &tx.SendTx{
		TxID:        *txID,
		Time:        txInfo.time.Unix(),
		BlockHeight: -1,
		Fee:         txInfo.fee,
		Receivers:   txInfo.outputs,
	}
	a.TxStore.Lock()
	a.TxStore.s = append(a.TxStore.s, sendtx)
	a.TxStore.dirty = true
	a.TxStore.Unlock()

	// Notify frontends of new SendTx.
	bs, err := GetCurBlock()
	if err == nil {
		for _, details := range sendtx.TxInfo(a.Name(), bs.Height, a.Net()) {
			NotifyNewTxDetails(frontendNotificationMaster, a.Name(),
				details)
		}
	}

	// Remove previous unspent outputs now spent by the tx.
	a.UtxoStore.Lock()
	modified := a.UtxoStore.s.Remove(txInfo.inputs)
	a.UtxoStore.dirty = a.UtxoStore.dirty || modified

	// Add unconfirmed change utxo (if any) to UtxoStore.
	if txInfo.changeUtxo != nil {
		a.UtxoStore.s = append(a.UtxoStore.s, txInfo.changeUtxo)
		a.ReqSpentUtxoNtfn(txInfo.changeUtxo)
		a.UtxoStore.dirty = true
	}
	a.UtxoStore.Unlock()

	// Disk sync tx and utxo stores.
	if err := a.writeDirtyToDisk(); err != nil {
		log.Errorf("cannot sync dirty wallet: %v", err)
	}

	// Notify all frontends of account's new unconfirmed and
	// confirmed balance.
	confirmed := a.CalculateBalance(1)
	unconfirmed := a.CalculateBalance(0) - confirmed
	NotifyWalletBalance(frontendNotificationMaster, a.name, confirmed)
	NotifyWalletBalanceUnconfirmed(frontendNotificationMaster, a.name, unconfirmed)

	// btcd cannot be trusted to successfully relay the tx to the
	// Bitcoin network.  Even if this succeeds, the rawtx must be
	// saved and checked for an appearence in a later block. btcd
	// will make a best try effort, but ultimately it's btcwallet's
	// responsibility.
	//
	// Add hex string of raw tx to sent tx pool.  If btcd disconnects
	// and is reconnected, these txs are resent.
	UnminedTxs.Lock()
	UnminedTxs.m[TXID(*txID)] = txInfo
	UnminedTxs.Unlock()

	log.Infof("Successfully sent transaction %v", result)
	ReplySuccess(frontend, icmd.Id(), result)

	// The comments to be saved differ based on the underlying type
	// of the cmd, so switch on the type to check whether it is a
	// SendFromCmd or SendManyCmd.
	//
	// TODO(jrick): If message succeeded in being sent, save the
	// transaction details with comments.
	switch cmd := icmd.(type) {
	case *btcjson.SendFromCmd:
		_ = cmd.Comment
		_ = cmd.CommentTo

	case *btcjson.SendManyCmd:
		_ = cmd.Comment
	}

	return true
}

// SetTxFee sets the transaction fee per kilobyte added to transactions.
func SetTxFee(frontend chan []byte, icmd btcjson.Cmd) {
	// Type assert icmd to access parameters.
	cmd, ok := icmd.(*btcjson.SetTxFeeCmd)
	if !ok {
		ReplyError(frontend, icmd.Id(), &btcjson.ErrInternal)
		return
	}

	// Check that amount is not negative.
	if cmd.Amount < 0 {
		e := &btcjson.Error{
			Code:    btcjson.ErrInvalidParams.Code,
			Message: "amount cannot be negative",
		}
		ReplyError(frontend, cmd.Id(), e)
		return
	}

	// Set global tx fee.
	TxFeeIncrement.Lock()
	TxFeeIncrement.i = cmd.Amount
	TxFeeIncrement.Unlock()

	// A boolean true result is returned upon success.
	ReplySuccess(frontend, cmd.Id(), true)
}

// CreateEncryptedWallet creates a new account with an encrypted
// wallet.  If an account with the same name as the requested account
// name already exists, an invalid account name error is returned to
// the client.
//
// Wallets will be created on TestNet3, or MainNet if btcwallet is run with
// the --mainnet option.
func CreateEncryptedWallet(frontend chan []byte, icmd btcjson.Cmd) {
	// Type assert icmd to access parameters.
	cmd, ok := icmd.(*btcws.CreateEncryptedWalletCmd)
	if !ok {
		ReplyError(frontend, icmd.Id(), &btcjson.ErrInternal)
		return
	}

	err := accountstore.CreateEncryptedWallet(cmd.Account, cmd.Description,
		[]byte(cmd.Passphrase))
	switch err {
	case nil:
		// A nil reply is sent upon successful wallet creation.
		ReplySuccess(frontend, cmd.Id(), nil)

	case ErrAcctNotExist:
		ReplyError(frontend, cmd.Id(),
			&btcjson.ErrWalletInvalidAccountName)

	case ErrBtcdDisconnected:
		e := &btcjson.Error{
			Code:    btcjson.ErrInternal.Code,
			Message: "btcd disconnected",
		}
		ReplyError(frontend, cmd.Id(), e)

	default:
		ReplyError(frontend, cmd.Id(), &btcjson.ErrInternal)
	}
}

// WalletIsLocked responds to the walletislocked extension request by
// replying with the current lock state (false for unlocked, true for
// locked) of an account.  An error is returned if the requested account
// does not exist.
func WalletIsLocked(frontend chan []byte, icmd btcjson.Cmd) {
	// Type assert icmd to access parameters.
	cmd, ok := icmd.(*btcws.WalletIsLockedCmd)
	if !ok {
		ReplyError(frontend, icmd.Id(), &btcjson.ErrInternal)
		return
	}

	a, err := accountstore.Account(cmd.Account)
	switch err {
	case nil:
		break

	case ErrAcctNotExist:
		ReplyError(frontend, cmd.Id(),
			&btcjson.ErrWalletInvalidAccountName)
		return

	default: // all other non-nil errors
		e := &btcjson.Error{
			Code:    btcjson.ErrWallet.Code,
			Message: err.Error(),
		}
		ReplyError(frontend, cmd.Id(), e)
		return
	}

	a.mtx.RLock()
	locked := a.Wallet.IsLocked()
	a.mtx.RUnlock()

	// Reply with true for a locked wallet, and false for unlocked.
	ReplySuccess(frontend, cmd.Id(), locked)
}

// WalletLock responds to walletlock request by locking the wallet,
// replying with an error if the wallet is already locked.
//
// TODO(jrick): figure out how multiple wallets/accounts will work
// with this.  Lock all the wallets, like if all accounts are locked
// for one bitcoind wallet?
func WalletLock(frontend chan []byte, icmd btcjson.Cmd) {
	a, err := accountstore.Account("")
	switch err {
	case nil:
		break

	case ErrAcctNotExist:
		e := &btcjson.Error{
			Code:    btcjson.ErrWallet.Code,
			Message: "default account does not exist",
		}
		ReplyError(frontend, icmd.Id(), e)
		return

	default: // all other non-nil errors
		e := &btcjson.Error{
			Code:    btcjson.ErrWallet.Code,
			Message: err.Error(),
		}
		ReplyError(frontend, icmd.Id(), e)
		return
	}

	switch err := a.Lock(); err {
	case nil:
		ReplySuccess(frontend, icmd.Id(), nil)

	default:
		ReplyError(frontend, icmd.Id(),
			&btcjson.ErrWalletWrongEncState)
	}
}

// WalletPassphrase responds to the walletpassphrase request by unlocking
// the wallet.  The decryption key is saved in the wallet until timeout
// seconds expires, after which the wallet is locked.
//
// TODO(jrick): figure out how to do this for non-default accounts.
func WalletPassphrase(frontend chan []byte, icmd btcjson.Cmd) {
	// Type assert icmd to access parameters.
	cmd, ok := icmd.(*btcjson.WalletPassphraseCmd)
	if !ok {
		ReplyError(frontend, icmd.Id(), &btcjson.ErrInternal)
		return
	}

	a, err := accountstore.Account("")
	switch err {
	case nil:
		break

	case ErrAcctNotExist:
		e := &btcjson.Error{
			Code:    btcjson.ErrWallet.Code,
			Message: "default account does not exist",
		}
		ReplyError(frontend, cmd.Id(), e)
		return

	default: // all other non-nil errors
		e := &btcjson.Error{
			Code:    btcjson.ErrWallet.Code,
			Message: err.Error(),
		}
		ReplyError(frontend, cmd.Id(), e)
		return
	}

	switch err := a.Unlock([]byte(cmd.Passphrase), cmd.Timeout); err {
	case nil:
		ReplySuccess(frontend, cmd.Id(), nil)

		go func(timeout int64) {
			time.Sleep(time.Second * time.Duration(timeout))
			_ = a.Lock()
		}(cmd.Timeout)

	case ErrAcctNotExist:
		ReplyError(frontend, cmd.Id(),
			&btcjson.ErrWalletInvalidAccountName)

	default:
		ReplyError(frontend, cmd.Id(),
			&btcjson.ErrWalletPassphraseIncorrect)
	}
}

// AccountNtfn is a struct for marshalling any generic notification
// about a account for a wallet frontend.
//
// TODO(jrick): move to btcjson so it can be shared with frontends?
type AccountNtfn struct {
	Account      string      `json:"account"`
	Notification interface{} `json:"notification"`
}

// NotifyWalletLockStateChange sends a notification to all frontends
// that the wallet has just been locked or unlocked.
func NotifyWalletLockStateChange(account string, locked bool) {
	ntfn := btcws.NewWalletLockStateNtfn(account, locked)
	mntfn, _ := ntfn.MarshalJSON()
	frontendNotificationMaster <- mntfn
}

// NotifyWalletBalance sends a confirmed account balance notification
// to a frontend.
func NotifyWalletBalance(frontend chan []byte, account string, balance float64) {
	ntfn := btcws.NewAccountBalanceNtfn(account, balance, true)
	mntfn, _ := ntfn.MarshalJSON()
	frontend <- mntfn
}

// NotifyWalletBalanceUnconfirmed sends a confirmed account balance
// notification to a frontend.
func NotifyWalletBalanceUnconfirmed(frontend chan []byte, account string, balance float64) {
	ntfn := btcws.NewAccountBalanceNtfn(account, balance, false)
	mntfn, _ := ntfn.MarshalJSON()
	frontend <- mntfn
}

// NotifyNewTxDetails sends details of a new transaction to a frontend.
func NotifyNewTxDetails(frontend chan []byte, account string,
	details map[string]interface{}) {

	ntfn := btcws.NewTxNtfn(account, details)
	mntfn, _ := ntfn.MarshalJSON()
	frontend <- mntfn
}
