package auth

type SignupRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type User struct {
	ID       string `json:"id"`
	Email    string `json:"email"`
	Password string `json:"-"`
	Role     string `json:"role"`
}

type TokenResponse struct {
	Token string `json:"token"`
}
