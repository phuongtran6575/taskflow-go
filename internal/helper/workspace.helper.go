package helper

import (
	"TaskFlow-Go/internal/dto"
	"TaskFlow-Go/internal/models"
	"TaskFlow-Go/internal/shared/apperror"
)

func IsDomainTaken(existingWorkspace *models.Workspace, excludeWorkspaceID string) bool {
	if existingWorkspace == nil {
		return false
	}
	return excludeWorkspaceID == "" || existingWorkspace.ID != excludeWorkspaceID
}

func AllWorkspacePlans() []models.WorkspacePlan {
	return []models.WorkspacePlan{
		models.WorkspacePlanFREE,
		models.WorkspacePlanPRO,
		models.WorkspacePlanENTERPRISE,
	}
}

// workspacePlanLimits định nghĩa giới hạn số workspace OWNER theo từng plan.
// -1 nghĩa là không giới hạn.
var workspacePlanLimits = map[models.WorkspacePlan]int64{
	models.WorkspacePlanFREE:       3,
	models.WorkspacePlanPRO:        10,
	models.WorkspacePlanENTERPRISE: -1,
}

// CheckWorkspacePlanLimit kiểm tra xem user đã đạt giới hạn số workspace
// với plan cho trước chưa. Trả về ErrWorkspaceLimitReached nếu vượt giới hạn.
func CheckWorkspacePlanLimit(planCounts []dto.WorkspacePlanCount, plan models.WorkspacePlan) error {
	limit, ok := workspacePlanLimits[plan]
	if !ok || limit == -1 {
		// Plan không xác định hoặc không giới hạn → cho phép
		return nil
	}

	// Tìm số lượng workspace hiện tại của plan này
	var current int64
	for _, pc := range planCounts {
		if pc.Plan == plan {
			current = pc.Count
			break
		}
	}

	if current >= limit {
		return apperror.ErrWorkspaceLimitReached
	}
	return nil
}
