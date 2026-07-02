package constant

import (
	"TaskFlow-Go/internal/models"
)

const (
	MaxWorkspaceLimit = 3

	WorkspaceOwner  models.WorkspaceRole = models.WorkspaceRoleOWNER
	WorkspaceAdmin  models.WorkspaceRole = models.WorkspaceRoleADMIN
	WorkspaceMember models.WorkspaceRole = models.WorkspaceRoleMEMBER

	WorkspacePlanFree       models.WorkspacePlan = models.WorkspacePlanFREE
	WorkspacePlanPro        models.WorkspacePlan = models.WorkspacePlanPRO
	WorkspacePlanEnterprise models.WorkspacePlan = models.WorkspacePlanENTERPRISE
)
