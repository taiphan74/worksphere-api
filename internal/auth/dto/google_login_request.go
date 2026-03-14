package dto

type GoogleLoginRequest struct {
	Email     string `json:"email" binding:"required,email"`
	FullName  string `json:"full_name"`
	AvatarURL string `json:"avatar_url"`
	IDToken   string `json:"id_token" binding:"required"`
}
