package models

type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=6"`
}

type LoginResponse struct {
	IDToken      string `json:"idToken"`
	Email        string `json:"email"`
	RefreshToken string `json:"refreshToken"`
	ExpiresIn    string `json:"expiresIn"`
	LocalID      string `json:"localId"`
}

type RegisterRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=6"`
}

type RegisterResponse struct {
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
	Age         int    `json:"age,omitempty" validate:"omitempty,min=0,max=150"`
	Address     string `json:"address,omitempty"`
	Gender      string `json:"gender,omitempty" validate:"omitempty,oneof=male female other"`
	PhoneNumber string `json:"phone_number,omitempty" validate:"omitempty,numeric,min=9,max=15"`
}
