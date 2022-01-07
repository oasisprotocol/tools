package main

import (
	"fmt"
	"os"
	"unicode"

	"github.com/pelletier/go-toml/v2"

	"github.com/oasisprotocol/oasis-core/go/common/quantity"
)

type Config struct {
	// DataDir is the base path where all of the faucet configuration data
	// and log files will live.
	DataDir string `toml:"data_dir"`

	// TargetAllowance is the target per-paratime allowance in base units.
	TargetAllowance quantity.Quantity `toml:"target_allowance"`
	// MaxConsensusFundAmount is the maximum amount of tokens funded to
	// consensus addresses in base units.
	MaxConsensusFundAmount quantity.Quantity `toml:"max_consensus_fund_amount"`
	// MaxParatimeFundAmount is the maximum amount of tokens funded to
	// paratime addresses in tokens.
	MaxParatimeFundAmount string `toml:"max_paratime_fund_amount"`

	// WebRoot is the base path where the static assets should be stored
	// and served from.
	WebRoot string `toml:"web_root"`
	// ListenAddr is the faucet RESTful API endpoint address.
	ListenAddr string `toml:"listen_addr"`
	// TLSCertFile is the TLS certificate file.
	TLSCertFile string `toml:"tls_cert_file"`
	// TLSKeyFile is the TLS certificate key file.
	TLSKeyFile string `toml:"tls_key_file"`
	// ReaptchaSharedSecret the reCAPTCHA V2 API shared secret for
	// use in bot prevention.
	RecaptchaSharedSecret string `toml:"recaptcha_shared_secret"`
}

func LoadConfig(path string) (*Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("cfg: failed to read configuration: %w", err)
	}

	var cfg Config
	if err = toml.Unmarshal(b, &cfg); err != nil {
		return nil, fmt.Errorf("cfg: failed to parse configuration: %w", err)
	}

	if cfg.DataDir == "" {
		return nil, fmt.Errorf("cfg: empty datadir")
	}
	if webRoot := cfg.WebRoot; webRoot != "" {
		fi, err := os.Stat(webRoot)
		if err != nil {
			return nil, fmt.Errorf("cfg: failed to stat webroot: %w", err)
		}
		if !fi.IsDir() {
			return nil, fmt.Errorf("cfg: webroot '%s' is not a directory", webRoot)
		}
	}
	if cfg.ListenAddr == "" {
		return nil, fmt.Errorf("cfg: empty listen addr")
	}
	if (cfg.TLSCertFile == "" && cfg.TLSKeyFile != "") || (cfg.TLSCertFile != "" && cfg.TLSKeyFile == "") {
		return nil, fmt.Errorf("cfg: both the TLS certificate and key must be provided")
	}
	if cfg.MaxParatimeFundAmount != "" {
		for _, c := range cfg.MaxParatimeFundAmount {
			if !unicode.IsDigit(c) {
				return nil, fmt.Errorf("cfg: max paratime fund amount is not a number")
			}
		}
	}

	return &cfg, nil
}
