package router

import (
	"TaskFlow-Go/internal/handler"
	"TaskFlow-Go/internal/middleware"

	"github.com/gin-gonic/gin"
)

type TaskRouter struct {
	handler           *handler.TaskHandler
	boardHandler      *handler.TaskBoardHandler
	assigneeHandler   *handler.TaskAssigneeHandler
	labelHandler      *handler.LabelHandler
	attachmentHandler *handler.AttachmentHandler
	commentHandler    *handler.CommentHandler
	mw                *middleware.Middleware
}

func NewTaskRouter(
	handler *handler.TaskHandler,
	boardHandler *handler.TaskBoardHandler,
	assigneeHandler *handler.TaskAssigneeHandler,
	labelHandler *handler.LabelHandler,
	attachmentHandler *handler.AttachmentHandler,
	commentHandler *handler.CommentHandler,
	mw *middleware.Middleware,
) *TaskRouter {
	return &TaskRouter{
		handler:           handler,
		boardHandler:      boardHandler,
		assigneeHandler:   assigneeHandler,
		labelHandler:      labelHandler,
		attachmentHandler: attachmentHandler,
		commentHandler:    commentHandler,
		mw:                mw,
	}
}

func (r *TaskRouter) RegisterRoutes(api *gin.RouterGroup) {
	auth := middleware.AuthMiddleware()

	notArchived := r.mw.RequireProjectNotArchived()

	// --- Board ---
	board := api.Group("/workspaces/:workspace_id/projects/:project_id", auth)
	{
		board.GET("/board", r.mw.RequireProjectPermission(middleware.PermTaskView), r.boardHandler.GetBoardData)
		board.GET("/board/columns/:column_id/tasks", r.mw.RequireProjectPermission(middleware.PermTaskView), r.boardHandler.LoadMoreTasksInColumn)
		board.PATCH("/tasks/:task_id/move", notArchived, r.mw.RequireProjectPermission(middleware.PermTaskMove), r.boardHandler.MoveTask)
	}

	// --- Tasks ---
	tasks := api.Group("/workspaces/:workspace_id/projects/:project_id/tasks", auth)
	{
		tasks.GET("/", r.mw.RequireProjectPermission(middleware.PermTaskView), r.handler.ListTasks)
		tasks.GET("/search", r.mw.RequireProjectPermission(middleware.PermTaskView), r.handler.SearchTasks)
		tasks.POST("/", notArchived, r.mw.RequireProjectPermission(middleware.PermTaskCreate), r.handler.CreateTask)
		tasks.GET("/:task_id", r.mw.RequireProjectPermission(middleware.PermTaskView), r.handler.GetTaskDetails)
		tasks.PATCH("/:task_id", notArchived, r.mw.RequireProjectPermission(middleware.PermTaskUpdate), r.handler.UpdateTask)
		tasks.DELETE("/:task_id", notArchived, r.mw.RequireProjectPermission(middleware.PermTaskDelete), r.handler.DeleteTask)
		tasks.POST("/:task_id/subtasks", notArchived, r.mw.RequireProjectPermission(middleware.PermTaskCreate), r.handler.CreateSubtask)
		tasks.GET("/:task_id/subtasks", r.mw.RequireProjectPermission(middleware.PermTaskView), r.handler.ListSubtasks)
	}

	// --- Assignees ---
	assignees := api.Group("/workspaces/:workspace_id/projects/:project_id/tasks/:task_id/assignees", auth)
	{
		assignees.GET("/", r.mw.RequireProjectPermission(middleware.PermTaskView), r.assigneeHandler.ListAssignees)
		assignees.GET("/available", r.mw.RequireProjectPermission(middleware.PermTaskAssign), r.assigneeHandler.GetAvailableAssignees)
		assignees.POST("/", notArchived, r.mw.RequireProjectPermission(middleware.PermTaskAssign), r.assigneeHandler.AssignMembers)
		assignees.DELETE("/", notArchived, r.mw.RequireProjectPermission(middleware.PermTaskAssign), r.assigneeHandler.UnassignMembers)
		assignees.POST("/me", notArchived, r.mw.RequireProjectMember(), r.assigneeHandler.SelfAssign)
		assignees.DELETE("/me", notArchived, r.mw.RequireProjectMember(), r.assigneeHandler.SelfUnassign)
	}

	// --- Labels ---
	labels := api.Group("/workspaces/:workspace_id/projects/:project_id/labels", auth)
	{
		labels.GET("/", r.mw.RequireProjectMember(), r.labelHandler.ListProjectLabels)
		labels.POST("/", notArchived, r.mw.RequireProjectPermission(middleware.PermLabelCreate), r.labelHandler.CreateLabel)
		labels.PATCH("/:label_id", notArchived, r.mw.RequireProjectPermission(middleware.PermLabelUpdate), r.labelHandler.UpdateLabel)
		labels.DELETE("/:label_id", notArchived, r.mw.RequireProjectPermission(middleware.PermLabelDelete), r.labelHandler.DeleteLabel)
	}

	// --- Task Labels ---
	taskLabels := api.Group("/workspaces/:workspace_id/projects/:project_id/tasks/:task_id/labels", auth)
	{
		taskLabels.GET("/", r.mw.RequireProjectMember(), r.labelHandler.ListTaskLabels)
		taskLabels.POST("/", notArchived, r.mw.RequireProjectPermission(middleware.PermLabelAssign), r.labelHandler.AssignLabels)
		taskLabels.DELETE("/", notArchived, r.mw.RequireProjectPermission(middleware.PermLabelAssign), r.labelHandler.RemoveLabels)
	}

	// --- Attachments ---
	attachments := api.Group("/workspaces/:workspace_id/projects/:project_id/tasks/:task_id/attachments", auth)
	{
		attachments.GET("/", r.mw.RequireProjectMember(), r.attachmentHandler.ListAttachments)
		attachments.POST("/", notArchived, r.mw.RequireProjectPermission(middleware.PermAttachmentUpload), r.attachmentHandler.UploadAttachments)
		attachments.GET("/:attachment_id/download", r.mw.RequireProjectMember(), r.attachmentHandler.GetDownloadUrl)
		attachments.GET("/:attachment_id/preview", r.mw.RequireProjectMember(), r.attachmentHandler.GetPreviewUrl)
		attachments.DELETE("/:attachment_id", notArchived, r.mw.RequireProjectPermission(middleware.PermAttachmentDeleteOwn, middleware.PermAttachmentDeleteAny), r.attachmentHandler.DeleteAttachment)
	}

	// --- Comments ---
	comments := api.Group("/workspaces/:workspace_id/projects/:project_id/tasks/:task_id/comments", auth)
	{
		comments.GET("/", r.mw.RequireProjectMember(), r.commentHandler.ListComments)
		comments.POST("/", notArchived, r.mw.RequireProjectPermission(middleware.PermCommentCreate), r.commentHandler.CreateComment)
		comments.PATCH("/:comment_id", notArchived, r.mw.RequireProjectPermission(middleware.PermCommentUpdateOwn), r.commentHandler.UpdateComment)
		comments.DELETE("/:comment_id", notArchived, r.mw.RequireProjectPermission(middleware.PermCommentDeleteOwn, middleware.PermCommentDeleteAny), r.commentHandler.DeleteComment)
		comments.GET("/mentionable", r.mw.RequireProjectMember(), r.commentHandler.GetMentionableUsers)
	}

	// --- Workspace-level routes ---
	myTasks := api.Group("/workspaces/:workspace_id", auth)
	{
		myTasks.GET("/my-tasks", r.mw.RequireWorkspaceRole(), r.handler.GetMyTasks)
	}

	storage := api.Group("/workspaces/:workspace_id", auth)
	{
		storage.GET("/storage", r.mw.RequireWorkspaceRole("OWNER", "ADMIN"), r.attachmentHandler.GetStorageUsage)
	}
}
