package workspacemodule

import (
	"github.com/jackc/pgx/v5/pgxpool"

	db "worksphere-api/internal/database/sqlc"
	"worksphere-api/internal/email"
	taskrepository "worksphere-api/internal/task/repository"
	workspacehandler "worksphere-api/internal/workspace/handler"
	workspacerepository "worksphere-api/internal/workspace/repository"
	workspaceservice "worksphere-api/internal/workspace/service"
)

// WorkspaceConfig chứa config riêng cho workspace module.
type WorkspaceConfig struct {
	InvitationFrontendURL string
	SMTPFrom              string
}

// WorkspaceDeps chứa dependency cho workspace handler setup.
type WorkspaceDeps struct {
	DBPool *pgxpool.Pool
}

// Setup khởi tạo workspace repository, service, handler.
func Setup(deps WorkspaceDeps) *workspacehandler.WorkspaceHandler {
	queries := db.New(deps.DBPool)
	workspaceRepo := workspacerepository.NewWorkspaceRepository(queries)
	memberRepo := workspacerepository.NewMemberRepository(queries)
	taskRepo := taskrepository.NewTaskRepository(queries)
	workspaceService := workspaceservice.NewWorkspaceService(deps.DBPool, workspaceRepo, memberRepo, taskRepo)
	return workspacehandler.NewWorkspaceHandler(workspaceService)
}

// MemberDeps chứa dependency cho member handler setup.
type MemberDeps struct {
	DBPool *pgxpool.Pool
}

// SetupMember khởi tạo member service, handler.
func SetupMember(deps MemberDeps) *workspacehandler.MemberHandler {
	queries := db.New(deps.DBPool)
	memberRepo := workspacerepository.NewMemberRepository(queries)
	memberService := workspaceservice.NewMemberService(deps.DBPool, memberRepo)
	return workspacehandler.NewMemberHandler(memberService)
}

// InvitationDeps chứa dependency cho invitation handler setup.
type InvitationDeps struct {
	DBPool       *pgxpool.Pool
	EmailService email.Service
	Config       WorkspaceConfig
}

// SetupInvitation khởi tạo invitation repository, service, handler.
func SetupInvitation(deps InvitationDeps) *workspacehandler.InvitationHandler {
	queries := db.New(deps.DBPool)
	invitationRepo := workspacerepository.NewInvitationRepository(queries)
	memberRepo := workspacerepository.NewMemberRepository(queries)
	workspaceRepo := workspacerepository.NewWorkspaceRepository(queries)
	invitationService := workspaceservice.NewInvitationService(
		deps.DBPool,
		invitationRepo,
		memberRepo,
		workspaceRepo,
		deps.EmailService,
		deps.Config.InvitationFrontendURL,
		deps.Config.SMTPFrom,
	)
	return workspacehandler.NewInvitationHandler(invitationService)
}