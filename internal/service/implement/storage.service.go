package implement

import (
	"errors"
	"fmt"
	"net/http"

	"gorm.io/gorm"

	"TaskFlow-Go/internal/dto"
	"TaskFlow-Go/internal/helper"
	repoInterface "TaskFlow-Go/internal/repository/interface"
	_interface "TaskFlow-Go/internal/service/interface"
	"TaskFlow-Go/internal/shared/apperror"
)

type storageService struct {
	attachmentRepo repoInterface.AttachmentRepository
	workspaceRepo  repoInterface.WorkspaceRepository
}

func NewStorageService(
	attachmentRepo repoInterface.AttachmentRepository,
	workspaceRepo repoInterface.WorkspaceRepository,
) _interface.StorageService {
	return &storageService{
		attachmentRepo: attachmentRepo,
		workspaceRepo:  workspaceRepo,
	}
}

func (s *storageService) GetWorkspaceStorageUsage(workspaceID string, userID string) (*dto.StorageUsageResponse, error) {
	workspace, err := s.workspaceRepo.GetByID(workspaceID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrWorkspaceNotFound
		}
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get workspace")
	}

	usageResult, err := s.attachmentRepo.GetStorageUsageByWorkspace(workspaceID)
	if err != nil {
		return nil, apperror.NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get storage usage")
	}

	limitBytes := helper.GetPlanLimits(workspace.Plan).MaxStorageBytes
	usedBytes := int64(0)
	for _, p := range usageResult.BreakdownByProject {
		usedBytes += p.UsedBytes
	}

	percentageUsed := 0
	if limitBytes > 0 {
		percentageUsed = int((usedBytes * 100) / limitBytes)
	}

	status := "OK"
	var warnings []string
	if usedBytes >= limitBytes {
		status = "EXCEEDED"
		warnings = append(warnings, "Workspace storage quota exceeded. Please upgrade your plan or free up space.")
	} else if percentageUsed > 95 {
		status = "CRITICAL"
		warnings = append(warnings, fmt.Sprintf("You have used %d%% of your storage quota (%s / %s). Please free up space.", percentageUsed, formatSizeDisplay(usedBytes), formatSizeDisplay(limitBytes)))
	} else if percentageUsed > 80 {
		status = "WARNING"
		warnings = append(warnings, fmt.Sprintf("You have used %d%% of your storage quota (%s / %s). Consider upgrading your plan.", percentageUsed, formatSizeDisplay(usedBytes), formatSizeDisplay(limitBytes)))
	}

	planName := string(workspace.Plan)
	if planName == "" {
		planName = "FREE"
	}

	return &dto.StorageUsageResponse{
		WorkspaceID: workspaceID,
		Plan:        planName,
		Storage: dto.StorageInfo{
			UsedBytes:      usedBytes,
			LimitBytes:     limitBytes,
			UsedDisplay:    formatSizeDisplay(usedBytes),
			LimitDisplay:   formatSizeDisplay(limitBytes),
			PercentageUsed: percentageUsed,
			Status:         status,
		},
		BreakdownByProject: usageResult.BreakdownByProject,
		Warnings:           warnings,
	}, nil
}
