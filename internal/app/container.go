package app

import (
	"TaskFlow-Go/internal/cache"
	"TaskFlow-Go/internal/database"
	"TaskFlow-Go/internal/handler"
	"TaskFlow-Go/internal/job"
	"TaskFlow-Go/internal/middleware"
	"TaskFlow-Go/internal/notif"
	repoImpl "TaskFlow-Go/internal/repository/implement"
	repoInterface "TaskFlow-Go/internal/repository/interface"
	"TaskFlow-Go/internal/router"
	serviceImpl "TaskFlow-Go/internal/service/implement"
	_interface "TaskFlow-Go/internal/service/interface"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type Container struct {
	DB *gorm.DB

	// Middleware provider
	Middleware *middleware.Middleware

	// Cache provider (in-memory, sau này có thể swap sang Redis)
	Cache cache.Provider

	// Notification dispatcher
	NotifDispatcher *notif.Dispatcher

	// Repositories
	UserRepo                  repoInterface.UserRepository
	SessionRepo               repoInterface.SessionRepository
	WorkspaceRepo             repoInterface.WorkspaceRepository
	PermissionRepo            repoInterface.PermissionRepository
	RoleRepo                  repoInterface.RoleRepository
	ProjectRepo               repoInterface.ProjectRepository
	ColumnRepo                repoInterface.ColumnRepository
	TaskRepo                  repoInterface.TaskRepository
	AttachmentRepo            repoInterface.AttachmentRepository
	CommentRepo               repoInterface.CommentRepository
	LabelRepo                 repoInterface.LabelRepository
	TaskAssigneeRepo          repoInterface.TaskAssigneeRepository
	TaskLabelRepo             repoInterface.TaskLabelRepository
	RolePermissionRepo        repoInterface.RolePermissionRepository
	ProjectMemberRepo         repoInterface.ProjectMemberRepository
	WorkspaceMemberRepo       repoInterface.WorkspaceMemberRepository
	WorkspaceInviteRepo       repoInterface.WorkspaceInviteRepository
	NotificationRepo          repoInterface.NotificationRepository
	ActivityLogRepo           repoInterface.ActivityLogRepository

	// Services
	AuthService            _interface.AuthService
	UserService            _interface.UserService
	SessionService         _interface.SessionService
	AvatarStorage          _interface.AvatarStorageService
	WorkspaceService       _interface.WorkspaceService
	WorkspaceMemberService _interface.WorkspaceMemberService
	WorkspaceInviteService _interface.WorkspaceInviteService
	PermissionService      _interface.PermissionService
	RoleService            _interface.RoleService
	ProjectService         _interface.ProjectService
	ProjectMemberService   _interface.ProjectMemberService
	ColumnService          _interface.ColumnService
	TaskService            _interface.TaskService
	TaskAssigneeService    _interface.TaskAssigneeService
	TaskBoardService       _interface.TaskBoardService
	LabelService           _interface.LabelService
	AttachmentService      _interface.AttachmentService
	CommentService         _interface.CommentService
	NotificationService    _interface.NotificationService
	StorageService         _interface.StorageService
	ActivityLogService     _interface.ActivityLogService

	// Handlers
	AuthHandler            *handler.AuthHandler
	UserHandler            *handler.UserHandler
	WorkspaceHandler       *handler.WorkspaceHandler
	WorkspaceMemberHandler *handler.WorkspaceMemberHandler
	WorkspaceInviteHandler *handler.WorkspaceInviteHandler
	PermissionHandler      *handler.PermissionHandler
	RoleHandler            *handler.RoleHandler
	ProjectHandler         *handler.ProjectHandler
	ProjectMemberHandler   *handler.ProjectMemberHandler
	ColumnHandler          *handler.ColumnHandler
	TaskHandler            *handler.TaskHandler
	TaskAssigneeHandler    *handler.TaskAssigneeHandler
	TaskBoardHandler       *handler.TaskBoardHandler
	LabelHandler           *handler.LabelHandler
	AttachmentHandler      *handler.AttachmentHandler
	CommentHandler         *handler.CommentHandler
	NotificationHandler    *handler.NotificationHandler
	ActivityLogHandler     *handler.ActivityLogHandler

	// Routers
	AuthRouter         *router.AuthRouter
	UserRouter         *router.UserRouter
	WorkspaceRouter    *router.WorkspaceRouter
	PermissionRouter   *router.PermissionRouter
	RoleRouter         *router.RoleRouter
	ProjectRouter      *router.ProjectRouter
	TaskRouter         *router.TaskRouter
	NotificationRouter *router.NotificationRouter
	ActivityLogRouter  *router.ActivityLogRouter
}

func NewContainer(db *gorm.DB) *Container {
	c := &Container{DB: db}

	tm := database.NewTransactionManager(db)

	// --- Initialize Repositories ---
	c.UserRepo = repoImpl.NewUserRepository(db)
	c.SessionRepo = repoImpl.NewSessionRepository(db)
	c.WorkspaceRepo = repoImpl.NewWorkspaceRepository(db)
	c.PermissionRepo = repoImpl.NewPermissionRepository(db)
	c.RoleRepo = repoImpl.NewRoleRepository(db)
	c.ProjectRepo = repoImpl.NewProjectRepository(db)
	c.ColumnRepo = repoImpl.NewColumnRepository(db)
	c.TaskRepo = repoImpl.NewTaskRepository(db)
	c.AttachmentRepo = repoImpl.NewAttachmentRepository(db)
	c.CommentRepo = repoImpl.NewCommentRepository(db)
	c.LabelRepo = repoImpl.NewLabelRepository(db)
	c.TaskAssigneeRepo = repoImpl.NewTaskAssigneeRepository(db)
	c.TaskLabelRepo = repoImpl.NewTaskLabelRepository(db)
	c.RolePermissionRepo = repoImpl.NewRolePermissionRepository(db)
	c.ProjectMemberRepo = repoImpl.NewProjectMemberRepository(db)
	c.WorkspaceMemberRepo = repoImpl.NewWorkspaceMemberRepository(db)
	c.WorkspaceInviteRepo = repoImpl.NewWorkspaceInviteRepository(db)
	c.NotificationRepo = repoImpl.NewNotificationRepository(db)
	c.ActivityLogRepo = repoImpl.NewActivityLogRepository(db)

	// --- Initialize Background Job Dispatcher ---
	dispatcher := job.NewDispatcher(db)

	// --- Initialize Middleware Provider ---
	c.Middleware = middleware.NewMiddleware(
		c.WorkspaceMemberRepo,
		c.WorkspaceRepo,
		c.ProjectMemberRepo,
		c.ProjectRepo,
	)

	// --- Initialize Cache Provider (BR-PERM-05) ---
	c.Cache = cache.NewMemoryCache()

	// --- Initialize Notification Dispatcher ---
	c.NotifDispatcher = notif.NewDispatcher(c.NotificationRepo)

	// --- Initialize Services ---
	c.AvatarStorage = serviceImpl.NewAvatarStorageService()
	c.AuthService = serviceImpl.NewAuthService(c.UserRepo)
	c.UserService = serviceImpl.NewUserService(c.UserRepo, c.SessionRepo, c.WorkspaceMemberRepo, c.AvatarStorage)
	c.SessionService = serviceImpl.NewSessionService(c.SessionRepo)
	c.WorkspaceService = serviceImpl.NewWorkspaceService(tm, c.WorkspaceRepo, c.RoleRepo, c.RolePermissionRepo, c.ActivityLogRepo, c.UserRepo, dispatcher)
	c.WorkspaceMemberService = serviceImpl.NewWorkspaceMemberService(c.WorkspaceMemberRepo, c.WorkspaceRepo, c.UserRepo, c.ProjectMemberRepo, tm, c.NotificationRepo, c.ActivityLogRepo)
	c.WorkspaceInviteService = serviceImpl.NewWorkspaceInviteService(c.WorkspaceInviteRepo, c.WorkspaceMemberRepo, c.WorkspaceRepo, c.NotificationRepo, c.UserRepo, c.NotifDispatcher)
	c.PermissionService = serviceImpl.NewPermissionService(c.PermissionRepo, c.Cache)
	c.RoleService = serviceImpl.NewRoleService(tm, c.RoleRepo, c.RolePermissionRepo, c.PermissionRepo, c.ProjectMemberRepo, c.WorkspaceRepo, c.ActivityLogRepo, c.UserRepo)
	c.ProjectService = serviceImpl.NewProjectService(tm, c.ProjectRepo, c.ColumnRepo, c.ProjectMemberRepo, c.WorkspaceRepo, c.RoleRepo, c.ActivityLogRepo, c.UserRepo, dispatcher)
	c.ProjectMemberService = serviceImpl.NewProjectMemberService(tm, c.ProjectMemberRepo, c.WorkspaceMemberRepo, c.WorkspaceRepo, c.ProjectRepo, c.RoleRepo, c.NotificationRepo, c.ActivityLogRepo, c.UserRepo, c.NotifDispatcher)
	c.ColumnService = serviceImpl.NewColumnService(tm, c.ColumnRepo, c.ProjectRepo, c.ActivityLogRepo, c.UserRepo)
	c.TaskService = serviceImpl.NewTaskService(tm, c.TaskRepo, c.TaskAssigneeRepo, c.TaskLabelRepo, c.ProjectRepo, c.ColumnRepo, c.ProjectMemberRepo, c.LabelRepo, c.WorkspaceRepo, c.ActivityLogRepo, c.NotificationRepo, c.UserRepo, c.NotifDispatcher)
	c.TaskAssigneeService = serviceImpl.NewTaskAssigneeService(c.TaskRepo, c.ProjectRepo, c.ProjectMemberRepo, c.TaskAssigneeRepo, c.WorkspaceRepo, c.NotificationRepo, c.ActivityLogRepo, c.UserRepo, c.NotifDispatcher, tm)
	c.TaskBoardService = serviceImpl.NewTaskBoardService(tm, c.TaskRepo, c.ColumnRepo, c.ProjectRepo, c.ProjectMemberRepo, c.TaskAssigneeRepo, c.TaskLabelRepo, c.ActivityLogRepo, c.NotificationRepo, c.UserRepo, c.NotifDispatcher)
	c.LabelService = serviceImpl.NewLabelService(tm, c.LabelRepo, c.TaskLabelRepo, c.ProjectRepo, c.TaskRepo, c.ActivityLogRepo, c.UserRepo)
	c.AttachmentService = serviceImpl.NewAttachmentService(tm, c.AttachmentRepo, c.TaskRepo, c.ProjectRepo, c.WorkspaceRepo, c.ActivityLogRepo, c.ProjectMemberRepo, c.UserRepo)
	c.CommentService = serviceImpl.NewCommentService(tm, c.CommentRepo, c.TaskRepo, c.ProjectRepo, c.ProjectMemberRepo, c.TaskAssigneeRepo, c.WorkspaceRepo, c.NotificationRepo, c.ActivityLogRepo, c.UserRepo, c.NotifDispatcher)
	c.NotificationService = serviceImpl.NewNotificationService(c.NotificationRepo, c.WorkspaceMemberRepo)
	c.StorageService = serviceImpl.NewStorageService(c.AttachmentRepo, c.WorkspaceRepo)
	c.ActivityLogService = serviceImpl.NewActivityLogService(c.ActivityLogRepo, c.ProjectRepo, c.TaskRepo, c.CommentRepo, c.WorkspaceRepo)

	// --- Initialize Handlers ---
	c.AuthHandler = handler.NewAuthHandler(c.AuthService)
	c.UserHandler = handler.NewUserHandler(c.UserService, c.SessionService)
	c.WorkspaceHandler = handler.NewWorkspaceHandler(c.WorkspaceService)
	c.WorkspaceMemberHandler = handler.NewWorkspaceMemberHandler(c.WorkspaceMemberService)
	c.WorkspaceInviteHandler = handler.NewWorkspaceInviteHandler(c.WorkspaceInviteService)
	c.PermissionHandler = handler.NewPermissionHandler(c.PermissionService)
	c.RoleHandler = handler.NewRoleHandler(c.RoleService)
	c.ProjectHandler = handler.NewProjectHandler(c.ProjectService)
	c.ProjectMemberHandler = handler.NewProjectMemberHandler(c.ProjectMemberService)
	c.ColumnHandler = handler.NewColumnHandler(c.ColumnService)
	c.TaskHandler = handler.NewTaskHandler(c.TaskService)
	c.TaskAssigneeHandler = handler.NewTaskAssigneeHandler(c.TaskAssigneeService)
	c.TaskBoardHandler = handler.NewTaskBoardHandler(c.TaskBoardService)
	c.LabelHandler = handler.NewLabelHandler(c.LabelService)
	c.AttachmentHandler = handler.NewAttachmentHandler(c.AttachmentService, c.StorageService)
	c.CommentHandler = handler.NewCommentHandler(c.CommentService)
	c.NotificationHandler = handler.NewNotificationHandler(c.NotificationService)
	c.ActivityLogHandler = handler.NewActivityLogHandler(c.ActivityLogService)

	// --- Initialize Routers ---
	c.AuthRouter = router.NewAuthRouter(c.AuthHandler)
	c.UserRouter = router.NewUserRouter(c.UserHandler)
	c.WorkspaceRouter = router.NewWorkspaceRouter(c.WorkspaceHandler, c.WorkspaceMemberHandler, c.WorkspaceInviteHandler, c.Middleware)
	c.PermissionRouter = router.NewPermissionRouter(c.PermissionHandler, c.Middleware)
	c.RoleRouter = router.NewRoleRouter(c.RoleHandler, c.Middleware)
	c.ProjectRouter = router.NewProjectRouter(c.ProjectHandler, c.ColumnHandler, c.ProjectMemberHandler, c.Middleware)
	c.TaskRouter = router.NewTaskRouter(c.TaskHandler, c.TaskBoardHandler, c.TaskAssigneeHandler, c.LabelHandler, c.AttachmentHandler, c.CommentHandler, c.Middleware)
	c.NotificationRouter = router.NewNotificationRouter(c.NotificationHandler, c.Middleware)
	c.ActivityLogRouter = router.NewActivityLogRouter(c.ActivityLogHandler, c.Middleware)

	return c
}

func (c *Container) SetupRoutes(api *gin.RouterGroup) {
	router.SetupRoutes(
		api,
		c.AuthRouter,
		c.UserRouter,
		c.WorkspaceRouter,
		c.PermissionRouter,
		c.RoleRouter,
		c.ProjectRouter,
		c.TaskRouter,
		c.NotificationRouter,
		c.ActivityLogRouter,
	)
}
