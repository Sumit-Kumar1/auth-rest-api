package models

// TokenResponse represents the JSON response structure for token-related operations.
// It contains the access and refresh tokens that will be sent to the client.
type TokenResponse struct {
	AccessToken  string `json:"token,omitempty"`
	RefreshToken string `json:"refresh_token,omitempty"`
}

// TokenData represents the internal token data structure.
// It contains both access and refresh token details including their IDs and expiration times.
type TokenData struct {
	// Access Token details
	AccessToken     string
	AccessID        string
	AccessExpiresAt int64
	// Refresh Token details
	RefreshToken     string
	RefreshID        string
	RefreshExpiresAt int64
}
