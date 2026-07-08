package implement

import (
	"errors"

	"gorm.io/gorm"

	"TaskFlow-Go/internal/dto"
	repoInterface "TaskFlow-Go/internal/repository/interface"
	_interface "TaskFlow-Go/internal/service/interface"
	"TaskFlow-Go/internal/shared/apperror"
)

type permissionService struct {
	permRepo repoInterface.PermissionRepository
}

func NewPermissionService(
	permRepo repoInterface.PermissionRepository,
) _interface.PermissionService {
	return &permissionService{
		permRepo: permRepo,
	}
}

func (s *permissionService) ListFlatPermissions(userID string) (*dto.PermissionFlatResponse, error) {
	permissions, err := s.permRepo.List()
	if err != nil {
		return nil, apperror.NewAppError(500, "INTERNAL_ERROR", "Failed to list permissions")
	}
	return &dto.PermissionFlatResponse{Data: permissions, Total: len(permissions)}, nil
}

func (s *permissionService) ListGroupedPermissions(userID string) (*dto.PermissionGroupedResponse, error) {
	permissions, err := s.permRepo.List()
	if err != nil {
		return nil, apperror.NewAppError(500, "INTERNAL_ERROR", "Failed to list permissions")
	}

	grouped := make(map[string][]dto.PermissionInfo)
	for _, p := range permissions {
		grouped[p.Module] = append(grouped[p.Module], p)
	}

	totalPerms := 0
	for _, v := range grouped {
		totalPerms += len(v)
	}

	return &dto.PermissionGroupedResponse{
		Data:             grouped,
		TotalModules:     len(grouped),
		TotalPermissions: totalPerms,
	}, nil
}

func (s *permissionService) ListModules(userID string) (*dto.ModuleListResponse, error) {
	moduleDescriptions := map[string]string{
		"task":       "Quản lý công việc trong project",
		"project":    "Cài đặt và quản lý dự án",
		"column":     "Quản lý cột trên Kanban board",
		"comment":    "Bình luận trong task",
		"label":      "Nhãn dán phân loại task",
		"attachment": "Tệp đính kèm trong task",
	}

	modules, err := s.permRepo.ListModules(moduleDescriptions)
	if err != nil {
		return nil, apperror.NewAppError(500, "INTERNAL_ERROR", "Failed to list modules")
	}
	return &dto.ModuleListResponse{Data: modules, Total: len(modules)}, nil
}

func (s *permissionService) GetPermissionsByModule(userID string, module string) (*dto.ModulePermissionsResponse, error) {
	result, err := s.permRepo.GetByModule(module)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &dto.ModulePermissionsResponse{Module: module, Data: []dto.PermissionInfo{}, Total: 0}, nil
		}
		return nil, apperror.NewAppError(500, "INTERNAL_ERROR", "Failed to get permissions")
	}
	return result, nil
}

func (s *permissionService) GetPermissionByIdOrSlug(userID string, idOrSlug string) (*dto.PermissionDetailResponse, error) {
	permission, err := s.permRepo.GetByIDOrSlug(idOrSlug)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrPermissionNotFound
		}
		return nil, apperror.NewAppError(500, "INTERNAL_ERROR", "Failed to get permission")
	}
	return permission, nil
}
