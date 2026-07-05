package middleware

import (
	repoInterface "TaskFlow-Go/internal/repository/interface"
)

type Middleware struct {
	workspaceMemberRepo repoInterface.WorkspaceMemberRepository
	workspaceRepo       repoInterface.WorkspaceRepository
	projectMemberRepo   repoInterface.ProjectMemberRepository
	projectRepo         repoInterface.ProjectRepository
}

func NewMiddleware(
	workspaceMemberRepo repoInterface.WorkspaceMemberRepository,
	workspaceRepo repoInterface.WorkspaceRepository,
	projectMemberRepo repoInterface.ProjectMemberRepository,
	projectRepo repoInterface.ProjectRepository,
) *Middleware {
	return &Middleware{
		workspaceMemberRepo: workspaceMemberRepo,
		workspaceRepo:       workspaceRepo,
		projectMemberRepo:   projectMemberRepo,
		projectRepo:         projectRepo,
	}
}
