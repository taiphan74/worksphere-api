package dto

import "worksphere-api/internal/user"

type MeResponse struct {
	User AuthUserData `json:"user"`
}

func NewMeResponse(user user.User) MeResponse {
	return MeResponse{
		User: NewAuthUserData(user),
	}
}
