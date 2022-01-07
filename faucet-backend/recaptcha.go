package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

const recaptchaAPIURL = "https://www.google.com/recaptcha/api/siteverify"

type RecaptchaV2Response struct {
	Success     bool     `json:"success"`
	ChallengeTS string   `json:"challenge_ts"`
	Hostname    string   `json:"hostname"`
	ErrorCodes  []string `json:"error-codes,omitempty"`
}

func (svc *Service) CheckRecaptcha(userResponse string) error {
	resp, err := http.PostForm(
		recaptchaAPIURL,
		url.Values{
			"secret":   {svc.cfg.RecaptchaSharedSecret},
			"response": {userResponse},
			// "remoteip" - Optional, so fuck Google.
		},
	)
	if err != nil {
		return fmt.Errorf("recaptcha: request failed: %w", err)
	}
	defer resp.Body.Close()

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("recaptcha: failed to read body: %w", err)
	}

	var apiResponse RecaptchaV2Response
	if err = json.Unmarshal(b, &apiResponse); err != nil {
		return fmt.Errorf("recaptcha: failed to parse response: %w", err)
	}

	if !apiResponse.Success {
		return fmt.Errorf("recaptcha: verification failed")
	}

	return nil
}
