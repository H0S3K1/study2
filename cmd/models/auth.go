package models

// AuthRequest is used for both Login and Register
type AuthRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=6"`
}

// AuthResponse is the unified response returned by your API to the client
type AuthResponse struct {
	IDToken      string `json:"idToken"`
	RefreshToken string `json:"refreshToken"`
	ExpiresIn    string `json:"expiresIn"`
	LocalID      string `json:"localId"` // This maps to user_id/localId
	Email        string `json:"email,omitempty"`
}

// Keep this Internal for Firebase decoding (since tags differ)
type RefreshResponse struct {
	ExpiresIn    string `json:"expires_in"`
	TokenType    string `json:"token_type"`
	RefreshToken string `json:"refresh_token"`
	IDToken      string `json:"id_token"`
	UserID       string `json:"localId"`
	ProjectID    string `json:"project_id"`
}

// SocialLoginRequest handles tokens from FB/Google/Apple
type SocialLoginRequest struct {
	IDToken     string `json:"idToken,omitempty"`
	AccessToken string `json:"accessToken,omitempty"`
	ProviderID  string `json:"providerId" validate:"required"`
}
type ProfileRequest struct {
	Name        string `json:"name,omitempty"`
	Age         int    `json:"age,omitempty" validate:"omitempty,min=0,max=100"`
	Address     string `json:"address,omitempty"`
	Gender      string `json:"gender,omitempty" validate:"omitempty,oneof=male female other"`
	PhoneNumber string `json:"phone_number,omitempty" validate:"omitempty,numeric,min=9,max=15"`
}
