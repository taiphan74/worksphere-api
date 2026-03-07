package dto

type UpdateUserRequest struct {
	Email    *string `json:"email,omitempty" binding:"omitempty,email"`
	FullName *string `json:"full_name,omitempty"`
}
