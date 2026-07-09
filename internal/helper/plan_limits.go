package helper

import (
	"TaskFlow-Go/internal/models"
	"TaskFlow-Go/internal/shared/apperror"
)

const (
	_1GB = 1 * 1024 * 1024 * 1024
	_20GB = 20 * _1GB
)

type ResourceLimits struct {
	MaxMembers               int64
	MaxProjects              int64
	MaxRoles                 int64
	MaxStorageBytes          int64
	AllowCustomRoles         bool
	ActivityLogRetentionDays int
	MaxSessionsPerUser       int
	MaxActiveInvites         int64 // BR-INV-03: Giới hạn invite link ACTIVE tối đa
}

var planLimits = map[models.WorkspacePlan]ResourceLimits{
	models.WorkspacePlanFREE: {
		MaxMembers:               5,
		MaxProjects:              3,
		MaxRoles:                 3,
		MaxStorageBytes:          _1GB,
		AllowCustomRoles:         false,
		ActivityLogRetentionDays: 7,
		MaxSessionsPerUser:       3,
		MaxActiveInvites:         3,
	},
	models.WorkspacePlanPRO: {
		MaxMembers:               50,
		MaxProjects:              -1,
		MaxRoles:                 20,
		MaxStorageBytes:          _20GB,
		AllowCustomRoles:         true,
		ActivityLogRetentionDays: 90,
		MaxSessionsPerUser:       10,
		MaxActiveInvites:         20,
	},
	models.WorkspacePlanENTERPRISE: {
		MaxMembers:               -1,
		MaxProjects:              -1,
		MaxRoles:                 -1,
		MaxStorageBytes:          -1,
		AllowCustomRoles:         true,
		ActivityLogRetentionDays: 365,
		MaxSessionsPerUser:       -1,
		MaxActiveInvites:         -1,
	},
}

func GetPlanLimits(plan models.WorkspacePlan) ResourceLimits {
	limits, ok := planLimits[plan]
	if !ok {
		return planLimits[models.WorkspacePlanFREE]
	}
	return limits
}

func CheckMemberLimit(plan models.WorkspacePlan, currentCount int64) error {
	limits := GetPlanLimits(plan)
	if limits.MaxMembers == -1 {
		return nil
	}
	if currentCount >= limits.MaxMembers {
		return apperror.ErrWorkspaceMemberLimitReached
	}
	return nil
}

func CheckProjectLimit(plan models.WorkspacePlan, currentCount int) error {
	limits := GetPlanLimits(plan)
	if limits.MaxProjects == -1 {
		return nil
	}
	if int64(currentCount) >= limits.MaxProjects {
		return apperror.ErrProjectLimitReached
	}
	return nil
}

func CheckStorageLimit(plan models.WorkspacePlan, currentBytes int64, newBytes int64) error {
	limits := GetPlanLimits(plan)
	if limits.MaxStorageBytes == -1 {
		return nil
	}
	if currentBytes+newBytes > limits.MaxStorageBytes {
		return apperror.ErrStorageQuotaExceeded
	}
	return nil
}

func CheckRoleLimit(plan models.WorkspacePlan, currentCount int) error {
	limits := GetPlanLimits(plan)
	if limits.MaxRoles == -1 {
		return nil
	}
	if int64(currentCount) >= limits.MaxRoles {
		return apperror.ErrRoleLimitReached
	}
	return nil
}

func CheckCustomRolesAllowed(plan models.WorkspacePlan) error {
	limits := GetPlanLimits(plan)
	if !limits.AllowCustomRoles {
		return apperror.ErrCustomRolesNotAllowed
	}
	return nil
}

func CheckInviteLimit(plan models.WorkspacePlan, currentActive int64) error {
	limits := GetPlanLimits(plan)
	if limits.MaxActiveInvites == -1 {
		return nil
	}
	if currentActive >= limits.MaxActiveInvites {
		return apperror.ErrInviteLinkLimitReached
	}
	return nil
}

func IsWorkspaceOverLimit(plan models.WorkspacePlan, memberCount int, projectCount int, storageUsedBytes int64) bool {
	limits := GetPlanLimits(plan)

	if limits.MaxMembers != -1 && int64(memberCount) > limits.MaxMembers {
		return true
	}
	if limits.MaxProjects != -1 && int64(projectCount) > limits.MaxProjects {
		return true
	}
	if limits.MaxStorageBytes != -1 && storageUsedBytes > limits.MaxStorageBytes {
		return true
	}
	return false
}
