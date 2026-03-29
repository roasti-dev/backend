package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

const (
	identityToolkitScope   = "https://www.googleapis.com/auth/cloud-platform"
	passwordPolicyEndpoint = "https://identitytoolkit.googleapis.com/admin/v2/projects/%s/config"
)

// PasswordPolicy holds the password constraints fetched from Firebase.
// Used both for local validation (Password VO) and is the single source of truth.
type PasswordPolicy struct {
	MinLength        int
	MaxLength        int
	RequireUppercase bool
	RequireNumeric   bool
}

// DefaultPasswordPolicy is used as fallback when the Firebase policy cannot be fetched
// (e.g. when running against the local emulator).
var DefaultPasswordPolicy = PasswordPolicy{
	MinLength:        8,
	MaxLength:        32,
	RequireUppercase: false,
	RequireNumeric:   false,
}

type firebaseConfigResponse struct {
	PasswordPolicyConfig *struct {
		Constraints *struct {
			RequireUppercase bool `json:"requireUppercase"`
			RequireNumeric   bool `json:"requireNumeric"`
			MinLength        int  `json:"minLength"`
			MaxLength        int  `json:"maxLength"`
		} `json:"constraints"`
	} `json:"passwordPolicyConfig"`
}

// GetPasswordPolicy fetches the current password policy from Firebase Identity Platform.
// credJSON is the service account credentials JSON; if nil, Application Default Credentials are used.
func GetPasswordPolicy(ctx context.Context, projectID string, credJSON []byte) (PasswordPolicy, error) {
	var tokenSource oauth2.TokenSource
	if len(credJSON) > 0 {
		creds, err := google.CredentialsFromJSONWithType(ctx, credJSON, google.ServiceAccount, identityToolkitScope)
		if err != nil {
			return PasswordPolicy{}, fmt.Errorf("parse credentials: %w", err)
		}
		tokenSource = creds.TokenSource
	} else {
		creds, err := google.FindDefaultCredentials(ctx, identityToolkitScope)
		if err != nil {
			return PasswordPolicy{}, fmt.Errorf("find default credentials: %w", err)
		}
		tokenSource = creds.TokenSource
	}
	httpClient := oauth2.NewClient(ctx, tokenSource)

	url := fmt.Sprintf(passwordPolicyEndpoint, projectID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return PasswordPolicy{}, fmt.Errorf("create request: %w", err)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return PasswordPolicy{}, fmt.Errorf("fetch config: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode != http.StatusOK {
		return PasswordPolicy{}, fmt.Errorf("firebase returned %d", resp.StatusCode)
	}

	var cfg firebaseConfigResponse
	if err := json.NewDecoder(resp.Body).Decode(&cfg); err != nil {
		return PasswordPolicy{}, fmt.Errorf("decode response: %w", err)
	}

	if cfg.PasswordPolicyConfig == nil || cfg.PasswordPolicyConfig.Constraints == nil {
		return DefaultPasswordPolicy, nil
	}

	c := cfg.PasswordPolicyConfig.Constraints
	return PasswordPolicy{
		MinLength:        c.MinLength,
		MaxLength:        c.MaxLength,
		RequireUppercase: c.RequireUppercase,
		RequireNumeric:   c.RequireNumeric,
	}, nil
}
