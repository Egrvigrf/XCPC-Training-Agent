package domain

type StateKV struct {
	Key int64   `json:"key"`
	Val float64 `json:"val"`
}

type User struct {
	Id       int    `json:"id;primary_key"`
	Name     string `json:"name"`
	Phone    string `json:"phone"`
	Password string `json:"password"`
	Status   int64  `json:"status"`
	IsSystem int64  `json:"is_system"`
	CreateAt int64  `json:"create_at"`
	UpdateAt int64  `json:"update_at"`
	DeleteAt int64  `json:"delete_at"`
}

type IdReq struct {
	Id int `json:"id" from:"id"`
}

type LoginReq struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginResp struct {
	Token    string `json:"token"`
	Id       int    `json:"id"`
	Name     string `json:"name"`
	Status   int64  `json:"status"`
	IsSystem int64  `json:"is_system"`
}

type RegisterReq struct {
	Name      string `json:"name"`
	Phone     string `json:"phone"`
	Password  string `json:"password"`
	Password2 string `json:"password2"`
}

type RegisterResp struct {
	Token  string `json:"token"`
	Id     int    `json:"id"`
	Name   string `json:"name"`
	Status int    `json:"status"`
}

type UserListReq struct {
	Ids   []int  `json:"ids,omitempty" query:"ids"`
	Name  string `json:"name,omitempty" query:"name"`
	Page  int    `json:"page,omitempty" query:"page"`
	Count int    `json:"count,omitempty" query:"count"`
}

type UserListResp struct {
	Count int64 `json:"count"`
	List  []*User
}

type UpPasswordReq struct {
	Id     int    `json:"id"`
	OldPwd string `json:"oldPwd"`
	NewPwd string `json:"newPwd"`
}
