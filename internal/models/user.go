package models

type User struct {
	Email       string `json:"email" firestore:"email"`
	Role        string `json:"role" firestore:"role"`
	Name        string `json:"name,omitempty" firestore:"name,omitempty"`
	Age         int    `json:"age,omitempty" firestore:"age,omitempty"`
	Address     string `json:"address,omitempty" firestore:"address,omitempty"`
	Balance     int    `json:"balance,omitempty" firestore:"balance,omitempty"`
	Gender      string `json:"gender,omitempty" firestore:"gender,omitempty"`
	PhoneNumber string `json:"phone_number,omitempty" firestore:"phone_number,omitempty"`
}
