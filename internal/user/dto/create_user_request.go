package dto

type CreateUserRequest struct {
	Email     string  `json:"email" binding:"required,email"`
	Password  string  `json:"password" binding:"required,min=8"`
	FullName  string  `json:"full_name" binding:"required"`
	Username  *string `json:"username,omitempty" binding:"omitempty,min=3,max=50"`
	AvatarURL *string `json:"avatar_url,omitempty" binding:"omitempty,url,max=500"`
	Phone     *string `json:"phone,omitempty" binding:"omitempty,max=20"`
	JobTitle  *string `json:"job_title,omitempty" binding:"omitempty,max=100"`
}
