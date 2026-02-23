package types

type RegisterReq struct {
	Email     string `json:"email" form:"email"`
	Password  string `json:"password" form:"password"`
	Username  string `json:"username" form:"username"`
	AvatarUrl string `json:"avatar_url" form:"avatar_url"`
	Rating    int    `json:"rating" form:"rating"`
}

type LoginReq struct {
	Email    string `json:"email" form:"email"`
	Password string `json:"password" form:"password"`
}

type UserInfo struct {
	ID        int64  `json:"id,string"`
	Email     string `json:"email"`
	Username  string `json:"username"`
	AvatarUrl string `json:"avatar_url"`
	Rating    int    `json:"rating"`
}

type LoginResp struct {
	Token string   `json:"token"`
	User  UserInfo `json:"user"`
}
