package dto

type UpdateUserRequest struct {
	FullName *string `json:"full_name"`
	Password *string `json:"password" binding:"omitempty,min=8,max=72"`
}
