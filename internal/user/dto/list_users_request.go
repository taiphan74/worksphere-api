package dto

type ListUsersRequest struct {
	Status string `form:"status"`
	Search string `form:"search"`
}
