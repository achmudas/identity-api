package user

type UserDTO struct {
	ID         int32  `json:"id"`
	Username   string `json:"username"`
	Email      string `json:"email"`
	AvatarLink string `json:"avatar"`
	Address    string `json:"address"`
}
