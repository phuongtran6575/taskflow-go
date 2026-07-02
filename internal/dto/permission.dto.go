package dto

type PermissionInfo struct {
	ID          string `json:"id"`
	Slug        string `json:"slug"`
	Module      string `json:"module"`
	Description string `json:"description"`
	IsSystem    bool   `json:"is_system"`
}

type ModuleInfo struct {
	Name             string `json:"name"`
	PermissionCount  int    `json:"permission_count"`
	Description      string `json:"description"`
}

type PermissionGroupedResponse struct {
	Data           map[string][]PermissionInfo `json:"data"`
	TotalModules   int                         `json:"total_modules"`
	TotalPermissions int                       `json:"total_permissions"`
}

type PermissionFlatResponse struct {
	Data  []PermissionInfo `json:"data"`
	Total int              `json:"total"`
}

type ModuleListResponse struct {
	Data  []ModuleInfo `json:"data"`
	Total int          `json:"total"`
}

type ModulePermissionsResponse struct {
	Module string          `json:"module"`
	Data   []PermissionInfo `json:"data"`
	Total  int              `json:"total"`
}

type PermissionDetailResponse struct {
	ID          string `json:"id"`
	Slug        string `json:"slug"`
	Module      string `json:"module"`
	Description string `json:"description"`
	IsSystem    bool   `json:"is_system"`
}
