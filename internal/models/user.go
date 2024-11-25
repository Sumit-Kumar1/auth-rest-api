package models

type UserReq struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type UserResp struct {
	Email string `json:"email"`
	Token string `json:"token"`
}
