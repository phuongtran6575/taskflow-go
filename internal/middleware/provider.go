package middleware

import (
	repoInterface "TaskFlow-Go/internal/repository/interface"
)

// Middleware là struct trung tâm chứa toàn bộ middleware có dependency.
// Được khởi tạo một lần trong container và inject vào các router cần dùng.
//
// Cách thêm middleware mới:
//  1. Thêm dependency (repo/service) vào struct này nếu chưa có
//  2. Viết method mới trả về gin.HandlerFunc
//  3. Gọi method đó trong router tương ứng
type Middleware struct {
	workspaceMemberRepo repoInterface.WorkspaceMemberRepository
	projectMemberRepo   repoInterface.ProjectMemberRepository
}

func NewMiddleware(
	workspaceMemberRepo repoInterface.WorkspaceMemberRepository,
	projectMemberRepo repoInterface.ProjectMemberRepository,
) *Middleware {
	return &Middleware{
		workspaceMemberRepo: workspaceMemberRepo,
		projectMemberRepo:   projectMemberRepo,
	}
}
