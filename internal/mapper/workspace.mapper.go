package mapper

import (
	"TaskFlow-Go/internal/dto"
	"TaskFlow-Go/internal/models"
)

func ToWorkspaceFromCreateWorkspaceRequest(req *dto.CreateWorkspaceRequest, ownerID string) *models.Workspace {
	return &models.Workspace{
		Domain:  req.Domain,
		Name:    req.Name,
		OwnerID: ownerID,
	}
}
