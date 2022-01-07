package main

import (
	"context"
	"time"

	consensus "github.com/oasisprotocol/oasis-core/go/consensus/api"
	consensusTx "github.com/oasisprotocol/oasis-core/go/consensus/api/transaction"
	staking "github.com/oasisprotocol/oasis-core/go/staking/api"

	"github.com/oasisprotocol/oasis-sdk/client-sdk/go/config"
	"github.com/oasisprotocol/oasis-sdk/client-sdk/go/connection"
	"github.com/oasisprotocol/oasis-sdk/client-sdk/go/modules/consensusaccounts"
	"github.com/oasisprotocol/oasis-sdk/client-sdk/go/types"
)

type FundRequest struct {
	ParaTime *config.ParaTime
	Account  *types.Address

	ConsensusAmount *types.Quantity
	ParaTimeAmount  *types.BaseUnits

	ResponseCh chan error
}

func (svc *Service) BankWorker() {
	svc.log.Printf("bank: started")

	// XXX: Wire into termination.
	ctx := context.Background()

	var (
		conn connection.Connection
		err  error
	)
	for {
		svc.log.Printf("bank: attempting to connect to gRPC endpoint")
		if conn, err = connection.Connect(ctx, svc.network); err != nil {
			svc.log.Printf("bank: failed to connect to node: %v", err)
			time.Sleep(15 * time.Second)
			continue
		}
		break
	}

	// Refill the allowances.
	svc.RefillAllowances(ctx, conn)

	svc.log.Printf("bank: connected to gRPC endpoint")

	// Mark as ready to accept requests.
	close(svc.readyCh)

	refillTicker := time.NewTicker(1 * time.Hour)
	for {
		select {
		case req := <-svc.fundRequestCh:
			// Note: Access control, validation, and non-debug logging is
			// handled by the frontend.
			if req.ParaTime == nil {
				svc.FundConsensusRequest(ctx, conn, req)
			} else {
				svc.FundParaTimeRequest(ctx, conn, req)
			}
		case <-refillTicker.C:
			svc.RefillAllowances(ctx, conn)
		case <-svc.quitCh:
			return
		}
	}
}

func (svc *Service) FundConsensusRequest(ctx context.Context, conn connection.Connection, req *FundRequest) {
	defer close(req.ResponseCh)

	xfer := staking.Transfer{
		To:     req.Account.ConsensusAddress(),
		Amount: *req.ConsensusAmount,
	}
	tx := staking.NewTransferTx(0, new(consensusTx.Fee), &xfer)
	if err := svc.SignAndSubmitConsensusTx(ctx, conn, tx); err != nil {
		// Logging is handled by the helper.
		req.ResponseCh <- err
		return
	}

	req.ResponseCh <- nil
}

func (svc *Service) FundParaTimeRequest(ctx context.Context, conn connection.Connection, req *FundRequest) {
	defer close(req.ResponseCh)

	// Just asssume that there is sufficient allowance, and that the periodic
	// refill adequately handles keeping the allowance topped off.

	tx := consensusaccounts.NewDepositTx(nil, &consensusaccounts.Deposit{
		To:     req.Account,
		Amount: *req.ParaTimeAmount,
	})
	if err := svc.SignAndSubmitMetaTx(ctx, conn, req.ParaTime, tx); err != nil {
		// Logging is handled by the helper.
		req.ResponseCh <- err
		return
	}

	req.ResponseCh <- nil
}

func (svc *Service) RefillAllowances(ctx context.Context, conn connection.Connection) {
	// Failures are ignored under the assumption that there is sufficient allowance
	// already.
	svc.log.Printf("bank: refilling allowances")

	// Query the existing allowances.
	consensusAccount, err := conn.Consensus().Staking().Account(ctx, &staking.OwnerQuery{
		Height: consensus.HeightLatest,
		Owner:  svc.address,
	})
	if err != nil {
		svc.log.Printf("bank: failed to query funding account: %v", err)
		return
	}

	for ptName, pt := range svc.network.ParaTimes.All {
		ptAddr := staking.NewRuntimeAddress(pt.Namespace())
		allowance := consensusAccount.General.Allowances[ptAddr]

		svc.log.Printf("refill: %v allowance: %v", ptName, allowance)

		// Figure out if we need to increase.
		toFund := svc.cfg.TargetAllowance.Clone()
		if err = toFund.Sub(&allowance); err != nil || toFund.IsZero() {
			svc.log.Printf("bank: paratime '%s' already has sufficient allowance: %v", ptName, allowance)
			continue
		}

		// Build the staking transaction.
		allow := staking.Allow{
			Beneficiary:  ptAddr,
			Negative:     false,
			AmountChange: *toFund,
		}
		tx := staking.NewAllowTx(0, new(consensusTx.Fee), &allow)
		if err := svc.SignAndSubmitConsensusTx(ctx, conn, tx); err != nil {
			svc.log.Printf("bank: failed to add allowance to paratime '%s': %v", ptName, err)
		}
	}
}
