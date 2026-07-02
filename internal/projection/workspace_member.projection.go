package projection

import "TaskFlow-Go/internal/models"

type MemberWithInfoRow struct {
	UserID   string
	FullName string
	Role     models.WorkspaceRole
}
