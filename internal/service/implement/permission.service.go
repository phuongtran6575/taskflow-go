package implement

import (
	"crypto/sha1"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"TaskFlow-Go/internal/cache"
	"TaskFlow-Go/internal/dto"
	repoInterface "TaskFlow-Go/internal/repository/interface"
	_interface "TaskFlow-Go/internal/service/interface"
	"TaskFlow-Go/internal/shared/apperror"
)

type permissionService struct {
	permRepo repoInterface.PermissionRepository
	cache    cache.Provider
}

const permissionsCacheKey = "permissions:all"
const permissionsCacheTTL = 24 * time.Hour

func NewPermissionService(
	permRepo repoInterface.PermissionRepository,
	cacheProvider cache.Provider,
) _interface.PermissionService {
	return &permissionService{
		permRepo: permRepo,
		cache:    cacheProvider,
	}
}

// getCachedPermissions lấy từ cache hoặc query DB + set cache.
func (s *permissionService) getCachedPermissions() ([]dto.PermissionInfo, error) {
	data, err := s.cache.GetOrSet(permissionsCacheKey, permissionsCacheTTL, func() ([]byte, error) {
		perms, err := s.permRepo.List()
		if err != nil {
			return nil, err
		}
		return json.Marshal(perms)
	})
	if err != nil {
		return nil, apperror.NewAppError(500, "INTERNAL_ERROR", "Failed to list permissions")
	}
	var perms []dto.PermissionInfo
	if err := json.Unmarshal(data, &perms); err != nil {
		return nil, apperror.NewAppError(500, "INTERNAL_ERROR", "Failed to unmarshal permissions")
	}
	return perms, nil
}

// computeETag tính SHA1 hash của permissions list để làm ETag.
func (s *permissionService) computeETag(perms []dto.PermissionInfo) string {
	h := sha1.New()
	for _, p := range perms {
		h.Write([]byte(p.ID))
		h.Write([]byte(p.Slug))
		h.Write([]byte(p.Module))
	}
	return fmt.Sprintf("\"%x\"", h.Sum(nil))
}

func (s *permissionService) GetPermissionsETag() string {
	perms, err := s.getCachedPermissions()
	if err != nil {
		return ""
	}
	return s.computeETag(perms)
}

func (s *permissionService) ListFlatPermissions(userID string) (*dto.PermissionFlatResponse, error) {
	permissions, err := s.getCachedPermissions()
	if err != nil {
		return nil, err
	}
	return &dto.PermissionFlatResponse{Data: permissions, Total: len(permissions)}, nil
}

func (s *permissionService) ListGroupedPermissions(userID string) (*dto.PermissionGroupedResponse, error) {
	permissions, err := s.getCachedPermissions()
	if err != nil {
		return nil, err
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
	allPerms, err := s.getCachedPermissions()
	if err != nil {
		return nil, err
	}

	var filtered []dto.PermissionInfo
	for _, p := range allPerms {
		if p.Module == module {
			filtered = append(filtered, p)
		}
	}
	if filtered == nil {
		filtered = []dto.PermissionInfo{}
	}
	return &dto.ModulePermissionsResponse{Module: module, Data: filtered, Total: len(filtered)}, nil
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
