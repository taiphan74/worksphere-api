package router

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	"worksphere-api/internal/config"
	"worksphere-api/internal/middleware"
	"worksphere-api/pkg/response"
)

type AuthHandler interface {
	Register(*gin.Context)
	Login(*gin.Context)
	VerifyEmail(*gin.Context)
	ResendVerification(*gin.Context)
	ForgotPassword(*gin.Context)
	ResetPassword(*gin.Context)
	GoogleLogin(*gin.Context)
	Me(*gin.Context)
	RefreshToken(*gin.Context)
}

type ProfileHandler interface {
	GetProfile(*gin.Context)
	UpdateProfile(*gin.Context)
	ChangePassword(*gin.Context)
	GetAvatarUploadURL(*gin.Context)
	ConfirmAvatarUpload(*gin.Context)
	GetAvatarViewURL(*gin.Context)
}

type WorkspaceHandler interface {
	CreateWorkspace(*gin.Context)
	ListWorkspaces(*gin.Context)
	GetWorkspaceByID(*gin.Context)
	GetWorkspaceBySlug(*gin.Context)
	UpdateWorkspace(*gin.Context)
	DeleteWorkspace(*gin.Context)
}

type WorkspaceMemberHandler interface {
	AddMember(*gin.Context)
	ListMembers(*gin.Context)
	GetMember(*gin.Context)
	UpdateMemberRole(*gin.Context)
	RemoveMember(*gin.Context)
}

type InvitationHandler interface {
	SendInvitation(*gin.Context)
	GetInvitation(*gin.Context)
	ListInvitations(*gin.Context)
	AcceptInvitation(*gin.Context)
	DeclineInvitation(*gin.Context)
	CancelInvitation(*gin.Context)
	ResendInvitation(*gin.Context)
}

type TaskHandler interface {
	CreateTask(*gin.Context)
	ListTasks(*gin.Context)
	GetTask(*gin.Context)
	UpdateTask(*gin.Context)
	DeleteTask(*gin.Context)
}

type UserRouteRegistrar interface {
	RegisterRoutes(*gin.RouterGroup)
}

type AdminUserRouteRegistrar interface {
	RegisterAdminRoutes(*gin.RouterGroup)
}

type Groups struct {
	Public    *gin.RouterGroup
	Protected *gin.RouterGroup
}

func New(cfg *config.Config, logger *slog.Logger, redisClient *redis.Client, middlewares ...gin.HandlerFunc) *gin.Engine {
	engine := gin.New()

	engine.Use(cors.New(cors.Config{
		AllowOrigins:     []string{cfg.FrontendOrigin},
		AllowMethods:     []string{"GET", "POST", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		AllowCredentials: true,
	}))

	engine.Use(middleware.RequestID())
	engine.Use(middleware.Logger(logger))
	engine.Use(middleware.Recovery(logger))

	api := engine.Group("/api")
	for _, handler := range middlewares {
		if handler != nil {
			api.Use(handler)
		}
	}
	api.GET("/health", func(c *gin.Context) {
		appStatus := "ok"
		redisStatus := "ok"
		statusCode := http.StatusOK
		ctx, cancel := context.WithTimeout(c.Request.Context(), time.Second)
		defer cancel()

		if err := redisClient.Ping(ctx).Err(); err != nil {
			appStatus = "degraded"
			redisStatus = "unavailable"
			statusCode = http.StatusServiceUnavailable
		}

		response.Success(c, statusCode, gin.H{
			"status": appStatus,
			"env":    cfg.AppEnv,
			"redis":  redisStatus,
		}, "success")
	})

	return engine
}

func NewGroups(engine *gin.Engine, authMiddleware gin.HandlerFunc) Groups {
	api := engine.Group("/api")
	public := api.Group("")
	protected := api.Group("")
	protected.Use(authMiddleware)

	return Groups{
		Public:    public,
		Protected: protected,
	}
}

func RegisterAuthRoutes(
	groups Groups,
	handler AuthHandler,
	registerIPMiddleware gin.HandlerFunc,
	loginIPMiddleware gin.HandlerFunc,
) {
	authPublic := groups.Public.Group("/auth")
	if registerIPMiddleware != nil {
		authPublic.POST("/register", registerIPMiddleware, handler.Register)
	} else {
		authPublic.POST("/register", handler.Register)
	}

	if loginIPMiddleware != nil {
		authPublic.POST("/login", loginIPMiddleware, handler.Login)
	} else {
		authPublic.POST("/login", handler.Login)
	}
	authPublic.GET("/verify-email", handler.VerifyEmail)
	authPublic.POST("/resend-verification", handler.ResendVerification)
	authPublic.POST("/forgot-password", handler.ForgotPassword)
	authPublic.POST("/reset-password", handler.ResetPassword)
	authPublic.POST("/google", handler.GoogleLogin)
	authPublic.POST("/refresh", handler.RefreshToken)

	authProtected := groups.Protected.Group("/auth")
	authProtected.GET("/me", handler.Me)
}

func RegisterUserRoutes(groups Groups, handler UserRouteRegistrar) {
	// Admin-only routes for user management
	adminGroup := groups.Protected.Group("/users")
	adminGroup.Use(middleware.RequireRoles("ADMIN", "SUPER_ADMIN"))
	handler.RegisterRoutes(adminGroup)
}

func RegisterProfileRoutes(groups Groups, handler ProfileHandler) {
	profile := groups.Protected.Group("/profile")
	profile.GET("", handler.GetProfile)
	profile.PATCH("", handler.UpdateProfile)
	profile.POST("/change-password", handler.ChangePassword)

	avatar := profile.Group("/avatar")
	avatar.POST("/upload-url", handler.GetAvatarUploadURL)
	avatar.POST("/confirm", handler.ConfirmAvatarUpload)
	avatar.GET("/view-url", handler.GetAvatarViewURL)
}

func RegisterWorkspaceRoutes(groups Groups, handler WorkspaceHandler) {
	workspaces := groups.Protected.Group("/workspaces")
	workspaces.POST("", handler.CreateWorkspace)
	workspaces.GET("", handler.ListWorkspaces)
	workspaces.GET("/:id", handler.GetWorkspaceByID)
	workspaces.GET("/slug/:slug", handler.GetWorkspaceBySlug)
	workspaces.PATCH("/:id", handler.UpdateWorkspace)
	workspaces.DELETE("/:id", handler.DeleteWorkspace)
}

func RegisterWorkspaceMemberRoutes(groups Groups, handler WorkspaceMemberHandler) {
	members := groups.Protected.Group("/workspaces/:id/members")
	members.POST("", handler.AddMember)
	members.GET("", handler.ListMembers)
	members.GET("/:userId", handler.GetMember)
	members.PATCH("/:userId", handler.UpdateMemberRole)
	members.DELETE("/:userId", handler.RemoveMember)
}

func RegisterInvitationRoutes(groups Groups, handler InvitationHandler) {
	// Workspace-based invitation management (for owners)
	invitations := groups.Protected.Group("/workspaces/:id/invitations")
	invitations.POST("", handler.SendInvitation)
	invitations.GET("", handler.ListInvitations)

	// Invitation actions (accept/decline/cancel/resend)
	invite := groups.Protected.Group("/invitations")
	invite.POST("/accept", handler.AcceptInvitation)
	invite.POST("/decline", handler.DeclineInvitation)
	invite.GET("/:invitationId", handler.GetInvitation)
	invite.DELETE("/:invitationId", handler.CancelInvitation)
	invite.POST("/:invitationId/resend", handler.ResendInvitation)
}

func RegisterTaskRoutes(groups Groups, handler TaskHandler) {
	tasks := groups.Protected.Group("/workspaces/:id/tasks")
	tasks.POST("", handler.CreateTask)
	tasks.GET("", handler.ListTasks)
	tasks.GET("/:taskId", handler.GetTask)
	tasks.PATCH("/:taskId", handler.UpdateTask)
	tasks.DELETE("/:taskId", handler.DeleteTask)
}
