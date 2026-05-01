package taskmodule

import (
	"github.com/jackc/pgx/v5/pgxpool"

	db "worksphere-api/internal/database/sqlc"
	taskhandler "worksphere-api/internal/task/handler"
	taskrepository "worksphere-api/internal/task/repository"
	taskservice "worksphere-api/internal/task/service"
	workspacerepository "worksphere-api/internal/workspace/repository"
)

// TaskDeps chứa dependency cho task module setup.
type TaskDeps struct {
	DBPool *pgxpool.Pool
}

// Setup khởi tạo task repository, service, handler.
// MemberRepo được tạo nội bộ vì task module cần nó cho authorization check.
func Setup(deps TaskDeps) *taskhandler.TaskHandler {
	queries := db.New(deps.DBPool)
	taskRepo := taskrepository.NewTaskRepository(queries)
	memberRepo := workspacerepository.NewMemberRepository(queries)
	tasksService := taskservice.NewTaskService(taskRepo, memberRepo)
	return taskhandler.NewTaskHandler(tasksService)
}