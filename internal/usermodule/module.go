package usermodule

import (
	"github.com/jackc/pgx/v5/pgxpool"

	db "worksphere-api/internal/database/sqlc"
	userhandler "worksphere-api/internal/user/handler"
	userrepository "worksphere-api/internal/user/repository"
	userservice "worksphere-api/internal/user/service"
)

// UserDeps chứa dependency cho user module setup.
type UserDeps struct {
	DBPool *pgxpool.Pool
}

// Setup khởi tạo user repository, service, handler.
func Setup(deps UserDeps) *userhandler.UserHandler {
	queries := db.New(deps.DBPool)
	userRepo := userrepository.NewUserRepository(queries)
	userService := userservice.NewUserService(userRepo)
	return userhandler.NewUserHandler(userService)
}