package models

type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=6"`
}

type RegisterRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=6"`
}

type AuthResponse struct {
	IDToken      string `json:"idToken"`
	Email        string `json:"email"`
	RefreshToken string `json:"refreshToken"`
	ExpiresIn    string `json:"expiresIn"`
	LocalID      string `json:"localId"`
}

type EditProfileRequest struct {
	Password string `json:"password" validate:"omitempty,min=6"`
	Email    string `json:"email" validate:"omitempty,email"`
}

type UpdateProfileRequest struct {
	Name        string `json:"name,omitempty"`
	Age         int    `json:"age,omitempty" validate:"omitempty,min=0,max=100"`
	Address     string `json:"address,omitempty"`
	Gender      string `json:"gender,omitempty" validate:"omitempty,oneof=male female other"`
	PhoneNumber string `json:"phone_number,omitempty" validate:"omitempty,numeric,min=9,max=15"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

type RefreshResponse struct {
	ExpiresIn    string `json:"expires_in"`
	TokenType    string `json:"token_type"`
	RefreshToken string `json:"refresh_token"`
	IDToken      string `json:"id_token"`
	UserID       string `json:"user_id"`
	ProjectID    string `json:"project_id"`
}

type SocialLoginRequest struct {
	IDToken     string `json:"idToken"`
	AccessToken string `json:"accessToken"`
	ProviderID  string `json:"providerId" validate:"required"`
}
