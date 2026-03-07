package dto

import db "worksphere-api/internal/database/sqlc"

type MeResponse struct {
	User AuthUserData `json:"user"`
}

func NewMeResponse(user db.User) MeResponse {
	return MeResponse{
		User: NewAuthUserData(user),
	}
}
