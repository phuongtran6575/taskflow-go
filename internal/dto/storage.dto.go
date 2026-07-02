package dto

type StorageInfo struct {
	UsedBytes      int64  `json:"used_bytes"`
	LimitBytes     int64  `json:"limit_bytes"`
	UsedDisplay    string `json:"used_display"`
	LimitDisplay   string `json:"limit_display"`
	PercentageUsed int    `json:"percentage_used"`
	Status         string `json:"status"`
}

type ProjectStorageBreakdown struct {
	ProjectID   string `json:"project_id"`
	ProjectName string `json:"project_name"`
	ProjectKey  string `json:"project_key"`
	UsedBytes   int64  `json:"used_bytes"`
	UsedDisplay string `json:"used_display"`
	FileCount   int    `json:"file_count"`
}

type StorageUsageResponse struct {
	WorkspaceID        string                    `json:"workspace_id"`
	Plan               string                    `json:"plan"`
	Storage            StorageInfo               `json:"storage"`
	BreakdownByProject []ProjectStorageBreakdown `json:"breakdown_by_project"`
	Warnings           []string                  `json:"warnings"`
}
