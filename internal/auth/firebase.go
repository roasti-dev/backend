package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type firebasePasswordSigner struct {
	apiKey          string
	identityBaseURL string
	tokenBaseURL    string
	client          *http.Client
}

func NewFirebasePasswordSigner(apiKey, identityBaseURL, tokenBaseURL string) FirebasePasswordSigner {
	return &firebasePasswordSigner{
		identityBaseURL: identityBaseURL,
		tokenBaseURL:    tokenBaseURL,
		apiKey:          apiKey,
		client:          &http.Client{},
	}
}

type firebaseAuthErrorResponse struct {
	Error struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Errors  []struct {
			Domain  string `json:"domain"`
			Reason  string `json:"reason"`
			Message string `json:"message"`
		} `json:"errors"`
	} `json:"error"`
}

func (f *firebasePasswordSigner) SignInWithPassword(ctx context.Context, email, password string) (SignInResult, error) {
	payload := map[string]any{
		"email":             email,
		"password":          password,
		"returnSecureToken": true,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return SignInResult{}, fmt.Errorf("marshal payload: %w", err)
	}
	url := f.identityBaseURL + ":signInWithPassword?key=" + f.apiKey

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return SignInResult{}, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := f.client.Do(req)
	if err != nil {
		return SignInResult{}, fmt.Errorf("signin request: %w", err)
	}
	defer resp.Body.Close() // nolint:errcheck

	if resp.StatusCode != http.StatusOK {
		var errResp firebaseAuthErrorResponse
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err != nil {
			return SignInResult{}, fmt.Errorf("decode error response: %w", err)
		}

		switch errResp.Error.Message {
		case "INVALID_PASSWORD", "EMAIL_NOT_FOUND":
			return SignInResult{}, ErrInvalidCredentials
		case "USER_DISABLED":
			return SignInResult{}, ErrUserDisabled
		default:
			return SignInResult{}, fmt.Errorf("firebase error: %s", errResp.Error.Message)
		}
	}

	var result struct {
		IDToken      string `json:"idToken"`
		RefreshToken string `json:"refreshToken"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return SignInResult{}, fmt.Errorf("decode response: %w", err)
	}

	return SignInResult{
		IDToken:      result.IDToken,
		RefreshToken: result.RefreshToken,
	}, nil
}

func (f *firebasePasswordSigner) RefreshToken(ctx context.Context, refreshToken string) (SignInResult, error) {
	payload := map[string]any{
		"grant_type":    "refresh_token",
		"refresh_token": refreshToken,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return SignInResult{}, fmt.Errorf("marshal payload: %w", err)
	}

	url := f.tokenBaseURL + "?key=" + f.apiKey
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return SignInResult{}, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := f.client.Do(req)
	if err != nil {
		return SignInResult{}, fmt.Errorf("refresh token request: %w", err)
	}
	defer resp.Body.Close() // nolint:errcheck

	if resp.StatusCode != http.StatusOK {
		var errResp firebaseAuthErrorResponse
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err != nil {
			return SignInResult{}, fmt.Errorf("decode error response: %w", err)
		}

		switch errResp.Error.Message {
		case "TOKEN_EXPIRED", "INVALID_REFRESH_TOKEN":
			return SignInResult{}, ErrInvalidRefreshToken
		case "USER_DISABLED":
			return SignInResult{}, ErrUserDisabled
		case "USER_NOT_FOUND":
			return SignInResult{}, ErrUserNotFound
		case "MISSING_REFRESH_TOKEN":
			return SignInResult{}, ErrMissingRefreshToken
		case "INVALID_GRANT_TYPE":
			return SignInResult{}, ErrInvalidGrantType
		default:
			return SignInResult{}, fmt.Errorf("firebase error: %s", errResp.Error.Message)
		}
	}

	var result struct {
		IDToken      string `json:"id_token"`
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return SignInResult{}, fmt.Errorf("decode response: %w", err)
	}

	return SignInResult{
		IDToken:      result.IDToken,
		RefreshToken: result.RefreshToken,
	}, nil
}
