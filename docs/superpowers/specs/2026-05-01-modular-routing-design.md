# Modular Routing — Tách DI Setup & Route Registration

## Mục tiêu

Tách logic DI setup và route registration đang gộp trong `server.go` + `router.go` ra từng domain module, giúp mỗi module tự chủ (self-contained) và dễ thêm/xóa/sửa.

## Vấn đề hiện tại

- `server.go`: Khởi tạo TẤT CẢ repo/service/handler của mọi domain (auth, user, profile, workspace, task) trong 1 hàm duy nhất
- `router.go`: Định nghĩa TẤT CẢ handler interface + TẤT CẢ `Register*Routes` functions trong 1 file duy nhất
- Thêm module mới phải sửa cả 2 file, dễ quên hoặc conflict

## Giải pháp

### Kiến trúc mới

Mỗi domain module có file `module.go` chứa:
- **`Deps` struct** — dependency cần thiết (DB pool, Redis, shared services, cross-module repos)
- **`Setup(deps) → Handler`** — khởi tạo repo → service → handler

Mỗi handler implement interface `RouteRegistrar` với method `RegisterRoutes(groups, middlewares...)`.

### Cấu trúc file

```
internal/auth/module.go
  ├── AuthDeps struct { ... }
  ├── AuthConfig struct { ... }
  └── Setup(deps AuthDeps) → *AuthHandler

internal/auth/handler/auth_handler.go
  └── func (h *AuthHandler) RegisterRoutes(groups router.Groups, middlewares ...gin.HandlerFunc)

internal/user/module.go
  ├── UserDeps struct { ... }
  └── Setup(deps UserDeps) → *UserHandler

internal/profile/module.go
  ├── ProfileDeps struct { ... }
  └── Setup(deps ProfileDeps) → *ProfileHandler

internal/workspace/module.go
  ├── WorkspaceDeps struct { ... }
  ├── MemberDeps struct { ... }
  ├── InvitationDeps struct { ... }
  ├── Setup(deps WorkspaceDeps) → *WorkspaceHandler
  ├── SetupMember(deps MemberDeps) → *MemberHandler
  └── SetupInvitation(deps InvitationDeps) → *InvitationHandler

internal/task/module.go
  ├── TaskDeps struct { ... }
  └── Setup(deps TaskDeps) → *TaskHandler
```

### RouteRegistrar interface

```go
// router/router.go — interface duy nhất
type RouteRegistrar interface {
    RegisterRoutes(groups Groups, middlewares ...gin.HandlerFunc)
}
```

Mỗi handler tự decide middleware nào apply cho route nào trong implementation của `RegisterRoutes`.

### Xóa bỏ từ router.go

- Tất cả handler interface: `AuthHandler`, `ProfileHandler`, `WorkspaceHandler`, `WorkspaceMemberHandler`, `InvitationHandler`, `TaskHandler`
- Tất cả `Register*Routes` functions
- `UserRouteRegistrar` và `AdminUserRouteRegistrar` interfaces

### Giữ lại trong router.go

- `New()` — tạo engine với middleware
- `NewGroups()` — tạo public/protected groups
- `Groups` struct
- `RouteRegistrar` interface (mới)

## Xử lý Dependency

### Shared dependencies

Khởi tạo 1 lần trong `server.go`, truyền vào module nào cần:

| Shared Service | Module sử dụng |
|---|---|
| `rateLimitService` | auth |
| `emailService` | auth, workspace (invitation) |
| `verificationService` | auth |
| `passwordResetService` | auth |
| `tokenManager` | auth |
| `r2Storage` | profile |

### Cross-module dependencies

Module nhận repo của module khác qua Deps struct trực tiếp:

| Module | Cross-module dependency |
|---|---|
| workspace | `taskRepo` (seed default tasks) |
| task | `memberRepo` (check quyền member) |
| invitation | `memberRepo` + `workspaceRepo` (check workspace + role) |

Không có circular dependency — module chỉ phụ thuộc repo interface, không phụ thuộc service của module khác.

## server.go sau refactor

```go
func SetupRouter(cfg *config.Config, logger *slog.Logger, dbPool *pgxpool.Pool, redisClient *redis.Client) (*gin.Engine, error) {
    // ── Shared Services ──
    rateLimitService := ratelimit.NewService(redisClient, logger)
    globalIPMiddleware := ratelimit.GlobalIPMiddleware(rateLimitService)
    emailService := email.NewSMTPService(cfg.SMTP)
    verificationService := verification.NewService(redisClient, ...)
    passwordResetService := verification.NewPasswordResetService(...)
    tokenManager := authjwt.NewManager(cfg.JWT)
    r2Storage, err := storage.NewR2Storage(cfg.R2)
    if err != nil {
        return nil, err
    }

    // ── Module Setup ──
    authHandler := authmodule.Setup(authmodule.AuthDeps{...})
    userHandler := usermodule.Setup(usermodule.UserDeps{...})
    profileHandler := profilemodule.Setup(profilemodule.ProfileDeps{...})
    workspaceHandler := workspacemodule.Setup(workspacemodule.WorkspaceDeps{...})
    memberHandler := workspacemodule.SetupMember(workspacemodule.MemberDeps{...})
    invitationHandler := workspacemodule.SetupInvitation(workspacemodule.InvitationDeps{...})
    taskHandler := taskmodule.Setup(taskmodule.TaskDeps{...})

    // ── Router + Route Registration ──
    engine := router.New(cfg, logger, redisClient, globalIPMiddleware)
    groups := router.NewGroups(engine, middleware.JWTAuth(tokenManager))

    registerIPMiddleware := ratelimit.RegisterIPMiddleware(rateLimitService)
    loginIPMiddleware := ratelimit.LoginIPMiddleware(rateLimitService)

    authHandler.RegisterRoutes(groups, registerIPMiddleware, loginIPMiddleware)
    userHandler.RegisterRoutes(groups)
    profileHandler.RegisterRoutes(groups)
    workspaceHandler.RegisterRoutes(groups)
    memberHandler.RegisterRoutes(groups)
    invitationHandler.RegisterRoutes(groups)
    taskHandler.RegisterRoutes(groups)

    return engine, nil
}
```

## Workspace module — 3 hàm Setup

Workspace domain tách thành 3 hàm setup vì chúng chia sẻ repo nhưng có service riêng:

- `Setup(deps WorkspaceDeps) → *WorkspaceHandler` — workspace CRUD
- `SetupMember(deps MemberDeps) → *MemberHandler` — member management
- `SetupInvitation(deps InvitationDeps) → *InvitationHandler` — invitation system

Cả 3 hàm nằm trong `internal/workspace/module.go`.

## Auth module — AuthConfig

Auth cần nhiều config riêng, tách thành `AuthConfig` struct trong Deps:

```go
type AuthConfig struct {
    EmailVerifyURL       string
    PasswordResetURL     string
    GoogleClientID       string
    RefreshExpiresInDays int
    AppEnv               string
    JWTExpiresInMinutes  int
}
```

## Thứ tự triển khai

1. Thêm `RouteRegistrar` interface vào router.go
2. Tạo `auth/module.go` — module đầu tiên,验证 pattern
3. Thêm `RegisterRoutes` method vào auth handler
4. Cập nhật server.go dùng auth module
5. Làm tương tự cho user, profile, task
6. Làm tương tự cho workspace (3 hàm Setup)
7. Dọn dẹp router.go — xóa interface + Register*Routes cũ
8. Chạy test, đảm bảo không break

## Tiêu chí thành công

- server.go chỉ còn orchestrate: shared service init → module setup → route registration
- router.go chỉ còn: New(), NewGroups(), Groups struct, RouteRegistrar interface
- Mỗi module tự chứa DI setup + route registration
- Thêm module mới chỉ cần: tạo module.go + handler.RegisterRoutes() + 2 dòng trong server.go
- Tất cả test hiện tại vẫn pass