package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/oasisprotocol/oasis-core/go/common/crypto/signature"
	fileSigner "github.com/oasisprotocol/oasis-core/go/common/crypto/signature/signers/file"
	staking "github.com/oasisprotocol/oasis-core/go/staking/api"

	"github.com/oasisprotocol/oasis-sdk/client-sdk/go/config"
)

// Note: As this is intended to be extremely simple, I am refraining from
// pulling in the typical mountain of fancy cli/service related dependencies.
//
// The primary design is to be a dumb RESTful endpoint that can move tokens
// from a pre-funded testnet address to consensus and paratime accounts.
//
// While this service will enforce minimal access control and an upper limit
// of tokens transfered per request, everything else is assumed to be handled
// by the consumer of the API.

type Service struct {
	cfg     *Config
	network *config.Network

	address staking.Address
	signer  signature.Signer

	log *log.Logger

	readyCh chan struct{}
	quitCh  chan struct{}
	doneCh  chan struct{}

	fundRequestCh chan *FundRequest
}

func NewService(cfg *Config) (*Service, error) {
	// Carve out the data directory.
	if err := os.MkdirAll(cfg.DataDir, 0o700); err != nil {
		return nil, fmt.Errorf("main: failed to create data dir: %w", err)
	}

	// Initialize logging.
	f, err := os.OpenFile(
		filepath.Join(cfg.DataDir, "faucet-backend.log"),
		os.O_APPEND|os.O_CREATE|os.O_RDWR,
		0o600,
	)
	if err != nil {
		return nil, fmt.Errorf("main: failed to open log file: %w", err)
	}

	// Load the signer.
	factory, err := fileSigner.NewFactory(cfg.DataDir, signature.SignerEntity)
	if err != nil {
		return nil, fmt.Errorf("main: failed to create signer factory: %w", err)
	}
	signer, err := factory.Load(signature.SignerEntity)
	if err != nil {
		return nil, fmt.Errorf("main: failed to load signer: %w", err)
	}

	return &Service{
		cfg:           cfg,
		network:       config.DefaultNetworks.All["testnet"], // Yes, this is hardcoded.
		address:       staking.NewAddress(signer.Public()),
		signer:        signer,
		log:           log.New(io.MultiWriter(os.Stdout, f), "", log.LstdFlags),
		readyCh:       make(chan struct{}),
		quitCh:        make(chan struct{}),
		doneCh:        make(chan struct{}),
		fundRequestCh: make(chan *FundRequest, 10),
	}, nil
}

func main() {
	cfgFile := flag.String("f", "faucet-backend.toml", "path to configuration file")
	flag.Parse()

	cfg, err := LoadConfig(*cfgFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "faucet-backend: failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	svc, err := NewService(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "faucet-backend: failed to initialize service: %v\n", err)
		os.Exit(1)
	}
	svc.log.Printf("service initialized: address: %s", svc.address)

	go svc.BankWorker()
	go svc.FrontendWorker()

	<-svc.doneCh
}
