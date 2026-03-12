package users

import "time"

type sessions struct {
	ID        string
	UserID    string
	Email     string
	CreatedAt time.Time
	ExpiresAt time.Time
}

type createSignupReq struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type User struct {
	ID       int
	Email    string
	Password string
}

type loginReq struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}