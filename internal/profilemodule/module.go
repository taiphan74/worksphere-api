package profilemodule

import (
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	db "worksphere-api/internal/database/sqlc"
	profilehandler "worksphere-api/internal/profile/handler"
	profilerepository "worksphere-api/internal/profile/repository"
	profileservice "worksphere-api/internal/profile/service"
	"worksphere-api/internal/storage"
)

// ProfileDeps chứa dependency cho profile module setup.
type ProfileDeps struct {
	DBPool         *pgxpool.Pool
	Storage        *storage.R2Storage
	AvatarUploadTTL time.Duration
	AvatarViewTTL   time.Duration
}

// Setup khởi tạo profile repository, service, handler.
func Setup(deps ProfileDeps) *profilehandler.ProfileHandler {
	queries := db.New(deps.DBPool)
	profileRepo := profilerepository.NewProfileRepository(queries)
	profileService := profileservice.NewProfileService(profileRepo, deps.Storage, deps.AvatarUploadTTL, deps.AvatarViewTTL)
	return profilehandler.NewProfileHandler(profileService)
}