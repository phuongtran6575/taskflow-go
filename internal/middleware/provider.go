package middleware

import (
	"TaskFlow-Go/internal/cache"
	repoInterface "TaskFlow-Go/internal/repository/interface"
)

type Middleware struct {
	workspaceMemberRepo repoInterface.WorkspaceMemberRepository
	workspaceRepo       repoInterface.WorkspaceRepository
	projectMemberRepo   repoInterface.ProjectMemberRepository
	projectRepo         repoInterface.ProjectRepository
	cache               cache.Provider
}

func NewMiddleware(
	workspaceMemberRepo repoInterface.WorkspaceMemberRepository,
	workspaceRepo repoInterface.WorkspaceRepository,
	projectMemberRepo repoInterface.ProjectMemberRepository,
	projectRepo repoInterface.ProjectRepository,
	cache cache.Provider,
) *Middleware {
	return &Middleware{
		workspaceMemberRepo: workspaceMemberRepo,
		workspaceRepo:       workspaceRepo,
		projectMemberRepo:   projectMemberRepo,
		projectRepo:         projectRepo,
		cache:               cache,
	}
}
