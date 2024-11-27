package models

type TokenResponse struct {
	AccessToken  string `json:"token,omitempty"`
	RefreshToken string `json:"refresh_token,omitempty"`
}

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
