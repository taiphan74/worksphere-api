package dto

type UpdateUserRequest struct {
	Email     *string `json:"email,omitempty" binding:"omitempty,email"`
	Password  *string `json:"password,omitempty" binding:"omitempty,min=8"`
	FullName  *string `json:"full_name,omitempty"`
	AvatarURL *string `json:"avatar_url,omitempty" binding:"omitempty,max=500"`
	Phone     *string `json:"phone,omitempty" binding:"omitempty,max=20"`
	JobTitle  *string `json:"job_title,omitempty" binding:"omitempty,max=100"`
	Status    *string `json:"status,omitempty" binding:"omitempty,max=20"`
}
