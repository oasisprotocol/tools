package main

import (
	"context"
	"fmt"

	consensusSignature "github.com/oasisprotocol/oasis-core/go/common/crypto/signature"
	consensus "github.com/oasisprotocol/oasis-core/go/consensus/api"
	consensusTx "github.com/oasisprotocol/oasis-core/go/consensus/api/transaction"

	"github.com/oasisprotocol/oasis-sdk/client-sdk/go/client"
	"github.com/oasisprotocol/oasis-sdk/client-sdk/go/config"
	"github.com/oasisprotocol/oasis-sdk/client-sdk/go/connection"
	"github.com/oasisprotocol/oasis-sdk/client-sdk/go/crypto/signature"
	"github.com/oasisprotocol/oasis-sdk/client-sdk/go/crypto/signature/ed25519"
	"github.com/oasisprotocol/oasis-sdk/client-sdk/go/modules/consensusaccounts"
	"github.com/oasisprotocol/oasis-sdk/client-sdk/go/types"
)

// One would think that the SDK would have nice helpers for doing this,
// since it is a common operation.  Instead pretend that we are DeFi DEX
// developers writing solidity, and copy-paste a shitload of code out of
// the cli.
//
// If I had known that I would need to do this, I would have named the
// binary along the lines of `ponzu-faucet` or `faucet-swap`.

func (svc *Service) SignAndSubmitConsensusTx(
	ctx context.Context,
	conn connection.Connection,
	tx *consensusTx.Transaction,
) error {
	// Query the current account nonce.  This in theory could be done once
	// and just incremented, but the faucet probably won't have enough load
	// to where this is a big deal.
	nonce, err := conn.Consensus().GetSignerNonce(ctx, &consensus.GetSignerNonceRequest{
		AccountAddress: svc.address,
		Height:         consensus.HeightLatest,
	})
	if err != nil {
		svc.log.Printf("tx/consensus: failed to query nonce: %v", err)
		return fmt.Errorf("failed to query nonce")
	}
	tx.Nonce = nonce

	// Estimate gas.
	gas, err := conn.Consensus().EstimateGas(ctx, &consensus.EstimateGasRequest{
		Signer:      svc.signer.Public(),
		Transaction: tx,
	})
	if err != nil {
		svc.log.Printf("tx/consensus: failed to estimate gas: %v", err)
		return fmt.Errorf("failed to estimate gas")
	}
	tx.Fee.Gas = gas

	// Sign the transaction.
	sigCtx := consensusSignature.Context([]byte(
		fmt.Sprintf("%s for chain %s", consensusTx.SignatureContext, svc.network.ChainContext),
	))
	signedTx, err := consensusSignature.SignSigned(svc.signer, sigCtx, tx)
	if err != nil {
		svc.log.Printf("tx/consensus: failed to sign transaction: %v", err)
		return fmt.Errorf("failed to sign transaction")
	}

	// Submit the transaction.
	if err = conn.Consensus().SubmitTx(
		ctx,
		&consensusTx.SignedTransaction{
			Signed: *signedTx,
		},
	); err != nil {
		svc.log.Printf("tx/consensus: failed to submit transaction: %v", err)
		return fmt.Errorf("failed to submit transaction")
	}

	return nil
}

func (svc *Service) SignAndSubmitMetaTx(
	ctx context.Context,
	conn connection.Connection,
	pt *config.ParaTime,
	tx *types.Transaction,
) error {
	// Query the current account nonce.
	nonce, err := conn.Runtime(pt).Accounts.Nonce(
		ctx,
		client.RoundLatest,
		types.NewAddressFromConsensus(svc.address),
	)
	if err != nil {
		svc.log.Printf("tx/meta: failed to query nonce: %v", err)
		return fmt.Errorf("failed to query nonce")
	}

	// Estimate gas.
	tx.AppendAuthSignature(
		types.NewSignatureAddressSpecEd25519(ed25519.PublicKey(svc.signer.Public())),
		nonce,
	)
	tx.AuthInfo.Fee.Gas, err = conn.Runtime(pt).Core.EstimateGas(
		ctx,
		client.RoundLatest,
		tx,
	)
	if err != nil {
		svc.log.Printf("tx/meta: failed to estimate gas: %v", err)
		return fmt.Errorf("failed to estimate gas")
	}

	// Sign the transaction.
	sigCtx := signature.DeriveChainContext(pt.Namespace(), svc.network.ChainContext)
	ts := tx.PrepareForSigning()
	if err := ts.AppendSign(sigCtx, ed25519.WrapSigner(svc.signer)); err != nil {
		svc.log.Printf("tx/meta: failed to sign transaction: %v", err)
		return fmt.Errorf("failed to sign transaction")
	}

	// WARNING: This is specialized to deposit transactions because
	// that is all we use this for.  This would have been a fully
	// generic function if it wasn't for this event nonsense.
	decoder := conn.Runtime(pt).ConsensusAccounts
	watchCtx, cancelFn := context.WithCancel(ctx)
	defer cancelFn()

	ch, err := conn.Runtime(pt).WatchEvents(watchCtx, []client.EventDecoder{decoder}, false)
	if err != nil {
		svc.log.Printf("tx/meta: failed to watch events: %v", err)
		return fmt.Errorf("failed to watch events")
	}

	resultCh := make(chan *consensusaccounts.DepositEvent)
	go func() {
		defer close(resultCh)
		defer cancelFn()

		expectedFrom := types.NewAddressFromConsensus(svc.address)
		expectedNonce := nonce

		for {
			var bev *client.BlockEvents
			select {
			// TODO: Timeout.
			case <-watchCtx.Done():
				return
			case bev = <-ch:
			}
			for _, ev := range bev.Events {
				ce, ok := ev.(*consensusaccounts.Event)
				if !ok || ce.Deposit == nil {
					continue
				}
				if !ce.Deposit.From.Equal(expectedFrom) || ce.Deposit.Nonce != expectedNonce {
					continue
				}
				resultCh <- ce.Deposit
				return
			}
		}
	}()

	// Submit the transaction.
	signedTx := ts.UnverifiedTransaction()
	meta, err := conn.Runtime(pt).SubmitTxMeta(ctx, signedTx)
	if err != nil {
		svc.log.Printf("tx/meta: failed to submit transaction: %v", err)
		return fmt.Errorf("failed to submit meta transaction")
	}
	if meta.CheckTxError != nil {
		svc.log.Printf("tx/meta: transaction check failed with error: module: %s code: %d message: %s",
			meta.CheckTxError.Module,
			meta.CheckTxError.Code,
			meta.CheckTxError.Message,
		)
		return fmt.Errorf("failed to check meta transaction")
	}

	// WARNING: Likewise this is specialized to deposit transactions.
	ev := <-resultCh
	if ev == nil {
		svc.log.Printf("tx/meta: failed to wait for event")
		return fmt.Errorf("failed to wait for event")
	}
	if !ev.IsSuccess() {
		svc.log.Printf("tx/meta: tx failed with error: module: %s code: %d",
			ev.Error.Module,
			ev.Error.Code,
		)
		return fmt.Errorf("transaction failed")
	}

	return nil
}
