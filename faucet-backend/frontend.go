package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/oasisprotocol/oasis-sdk/client-sdk/go/helpers"
	"github.com/oasisprotocol/oasis-sdk/client-sdk/go/types"
)

const (
	queryParaTime          = "paratime"
	queryAccount           = "account"
	queryAmount            = "amount"
	queryRecaptchaResponse = "g-recaptcha-response"

	prefixOasis = "oasis"
	prefixEth   = "0x"
)

func (svc *Service) FrontendWorker() {
	defer func() {
		close(svc.doneCh)
	}()

	svc.log.Printf("frontend: started")

	// Register API endpoints.
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/fund", svc.OnFundRequest)
	if svc.cfg.WebRoot != "" {
		mux.Handle("/", http.FileServer(http.Dir(svc.cfg.WebRoot)))
	}

	srv := &http.Server{
		Addr:    svc.cfg.ListenAddr,
		Handler: mux,
	}

	// Wait till the part that does the actual heavy lifting is initialized.
	<-svc.readyCh

	svc.log.Printf("frontend: bank ready, starting HTTP server")

	// Serve.
	go func() {
		defer close(svc.quitCh)
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
		<-sigCh

		svc.log.Printf("frontend: user requested termination")

		if err := srv.Shutdown(context.Background()); err != nil {
			svc.log.Printf("frontend: failed graceful HTTP server shutdown: %v", err)
		}
	}()
	switch {
	case svc.cfg.TLSCertFile != "" || svc.cfg.TLSKeyFile != "":
		if err := srv.ListenAndServeTLS(svc.cfg.TLSCertFile, svc.cfg.TLSKeyFile); err != http.ErrServerClosed {
			svc.log.Printf("frontend: failed to start HTTPs server: %v", err)
			return
		}
	default:
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			svc.log.Printf("frontend: failed to start HTTP server: %v", err)
			return
		}
	}

	// Wait till all pending requests have been serviced.
	<-svc.quitCh
}

// onFundRequest handles a funding request.  The expect request is a POST of
// the form `https://host:port/api/v1/fund&account=CONSENSUS_ACCOUNT_ID&amount=TOKENS`.
func (svc *Service) OnFundRequest(w http.ResponseWriter, req *http.Request) {
	writeResult := func(statusCode int, result error) {
		type fundResponse struct {
			Result string `json:"result"`
		}

		w.WriteHeader(statusCode)
		b, _ := json.Marshal(&fundResponse{
			Result: result.Error(),
		})
		_, _ = w.Write(b)
	}

	// Ensure the user is POSTing, if auth is enabled.
	authEnabled := svc.cfg.RecaptchaSharedSecret != ""
	if authEnabled {
		if req.Method != http.MethodPost {
			svc.log.Printf("frontend: invalid http method: '%v'", req.Method)
			writeResult(
				http.StatusMethodNotAllowed,
				fmt.Errorf("invalid http method: '%v'", req.Method),
			)
			return
		}
	}

	// Parse the query and POST form (combined).
	if err := req.ParseForm(); err != nil {
		svc.log.Printf("frontend: invalid http request: %v", err)
		writeResult(
			http.StatusBadRequest,
			fmt.Errorf("invalid http request, failed to parse query/form"),
		)
		return
	}

	var err error
	fundReq := &FundRequest{
		ResponseCh: make(chan error),
	}

	// ParaTime
	paraTimeStr := strings.TrimSpace(req.Form.Get(queryParaTime))
	if paraTimeStr != "" {
		fundReq.ParaTime = svc.network.ParaTimes.All[paraTimeStr]
		if fundReq.ParaTime == nil {
			svc.log.Printf("frontend: invalid paratime: '%v'", paraTimeStr)
			writeResult(
				http.StatusInternalServerError,
				fmt.Errorf("failed to fund account: invalid paratime: '%v'", paraTimeStr),
			)
			return
		}
	}

	// Account
	accountStr := strings.TrimSpace(req.Form.Get(queryAccount))
	switch {
	case fundReq.ParaTime == nil && !strings.HasPrefix(accountStr, prefixOasis):
		svc.log.Printf("frontend: account not an oasis address: '%v'", accountStr)
		writeResult(
			http.StatusInternalServerError,
			fmt.Errorf("failed to fund account: invalid account: not an oasis address"),
		)
		return
	case fundReq.ParaTime != nil && !strings.HasPrefix(accountStr, prefixEth):
		// XXX: Does cipher use ethereum style `0x` prefixes?
		svc.log.Printf("frontend: account not an ethereum address: '%v'", accountStr)
		writeResult(
			http.StatusInternalServerError,
			fmt.Errorf("failed to fund account: invalid account: not an ethereum address"),
		)
		return
	}
	if fundReq.Account, err = helpers.ResolveAddress(svc.network, accountStr); err != nil {
		svc.log.Printf("frontend: invalid account '%v': %v", accountStr, err)
		writeResult(
			http.StatusInternalServerError,
			fmt.Errorf("failed to fund account: invalid account: '%v'", accountStr),
		)
		return
	}

	// Amount
	amountStr := strings.TrimSpace(req.Form.Get(queryAmount))
	switch fundReq.ParaTime {
	case nil:
		if fundReq.ConsensusAmount, err = helpers.ParseConsensusDenomination(
			svc.network,
			amountStr,
		); err != nil {
			svc.log.Printf("frontend: invalid amount '%v': %v", amountStr, err)
			writeResult(
				http.StatusInternalServerError,
				fmt.Errorf("failed to fund account: invalid amount: '%v'", amountStr),
			)
			return
		}
		if !svc.cfg.MaxConsensusFundAmount.IsZero() {
			max := svc.cfg.MaxConsensusFundAmount.Clone()
			if err = max.Sub(fundReq.ConsensusAmount); err != nil {
				svc.log.Printf("frontend: excessive consensus amount: %v", fundReq.ConsensusAmount)
				writeResult(
					http.StatusInternalServerError,
					fmt.Errorf("failed to fund account: excessive consensus amount: '%v'", amountStr),
				)
				return
			}
		}
	default:
		if fundReq.ParaTimeAmount, err = helpers.ParseParaTimeDenomination(
			fundReq.ParaTime,
			amountStr,
			types.NativeDenomination, // XXX: Make this configurable.
		); err != nil {
			svc.log.Printf("frontend: invalid amount '%v': %v", amountStr, err)
			writeResult(
				http.StatusInternalServerError,
				fmt.Errorf("failed to fund account: invalid amount: '%v'", amountStr),
			)
			return
		}
		if maxStr := svc.cfg.MaxParatimeFundAmount; maxStr != "" {
			max, err := helpers.ParseParaTimeDenomination(
				fundReq.ParaTime,
				maxStr,
				types.NativeDenomination,
			)
			if err != nil {
				svc.log.Printf("frontend: invalid maximum amount '%v': %v", maxStr, err)
				writeResult(
					http.StatusInternalServerError,
					fmt.Errorf("failed to fund account: per-paratime max misconfigured"),
				)
				return
			}
			if err = max.Amount.Sub(&fundReq.ParaTimeAmount.Amount); err != nil {
				svc.log.Printf("frontend: excessive paratime amount: %v", fundReq.ParaTimeAmount)
				writeResult(
					http.StatusInternalServerError,
					fmt.Errorf("failed to fund account: excessive paratime amount: '%v'", amountStr),
				)
				return
			}
		}
	}

	// Handle reCAPTCHA integration, if enabled.
	if authEnabled {
		// Technically not a query, but the server has a unified view of
		// POST form and query fields.
		if err = svc.CheckRecaptcha(req.Form.Get(queryRecaptchaResponse)); err != nil {
			svc.log.Printf("frontend: reCAPTCHA failed: %v", err)
			writeResult(
				http.StatusForbidden,
				fmt.Errorf("failed to verify reCAPTCHA"),
			)
			return
		}
	}

	// Attempt to fund the address.
	svc.fundRequestCh <- fundReq

	// Return the status.
	if err = <-fundReq.ResponseCh; err != nil {
		svc.log.Printf("frontend: failed to fund request ([%v]%v: %v): %v", paraTimeStr, accountStr, amountStr, err)
		writeResult(
			http.StatusInternalServerError,
			fmt.Errorf("failed to fund account: %w", err),
		)
		return
	}

	svc.log.Printf("frontend: request funded: [%v]%v: %v TEST", paraTimeStr, accountStr, amountStr)

	writeResult(
		http.StatusOK,
		fmt.Errorf("funding successful"),
	)
}
