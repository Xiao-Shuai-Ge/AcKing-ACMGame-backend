package types

type RegisterReq struct {
	Email    string `json:"email" form:"email"`
	Code     string `json:"code" form:"code"`
	Password string `json:"password" form:"password"`
	Username string `json:"username" form:"username"`
}

type SendCodeReq struct {
	Email string `json:"email" form:"email"`
}

type SendCodeResp struct {
}

type LoginReq struct {
	Email    string `json:"email" form:"email"`
	Password string `json:"password" form:"password"`
}

type UserInfo struct {
	ID       int64  `json:"id,string"`
	Email    string `json:"email"`
	Username string `json:"username"`
	Rating   int    `json:"rating"`
}

type LoginResp struct {
	Token string   `json:"token"`
	User  UserInfo `json:"user"`
}

type GetProfileReq struct {
	UserID int64 `json:"-" form:"-"`
}

type GetProfileResp struct {
	ID       int64  `json:"id,string"`
	Email    string `json:"email"`
	Username string `json:"username"`
	Rating   int    `json:"rating"`
}

type UpdateProfileReq struct {
	UserID   int64  `json:"-" form:"-"`
	Username string `json:"username" form:"username"`
}

type UpdateProfileResp struct {
}

type GetUserInfoReq struct {
	UserID int64 `json:"user_id" form:"user_id"`
}

type GetUserInfoResp struct {
	UserInfo
}
