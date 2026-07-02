package _interface

import (
	"TaskFlow-Go/internal/dto"
	"TaskFlow-Go/internal/models"
	"TaskFlow-Go/internal/projection"
)

type WorkspaceMemberRepository interface {
	Create(member *models.WorkspaceMember) error
	GetByID(workspaceID, userID string) (*models.WorkspaceMember, error)
	ListByUserID(userID string) ([]models.WorkspaceMember, error)
	UpdateRole(workspaceID, userID string, role models.WorkspaceRole) error
	Delete(workspaceID, userID string) error

	ListWithPagination(workspaceID string, search string, role string, page int, limit int) ([]dto.MemberInfo, *dto.Pagination, error)
	GetByIDWithDetails(workspaceID string, userID string) (*dto.MemberDetailResponse, error)
	GetMemberWithInfor(workspaceID string, userID string) (*projection.MemberWithInfoRow, error)
	GetUserProjectIDsInWorkspace(workspaceID, userID string) ([]string, error)
	CountMembers(workspaceID string) (int64, error)
}
