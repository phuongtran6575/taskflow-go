package dto

type CreateLabelRequest struct {
	Name  string `json:"name" binding:"required"`
	Color string `json:"color" binding:"required"`
}

type UpdateLabelRequest struct {
	Name  *string `json:"name,omitempty" binding:"omitempty,min=1"`
	Color *string `json:"color,omitempty"`
}

type AssignLabelsRequest struct {
	LabelIDs []string `json:"label_ids" binding:"required"`
}

type RemoveLabelsRequest struct {
	LabelIDs []string `json:"label_ids" binding:"required"`
}

type LabelInfo struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Color     string `json:"color"`
	TaskCount int    `json:"task_count,omitempty"`
	CreatedAt string `json:"created_at,omitempty"`
	UpdatedAt string `json:"updated_at,omitempty"`
}

type LabelListResponse struct {
	Data  []LabelInfo `json:"data"`
	Total int         `json:"total"`
}

type LabelCreateResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Color     string `json:"color"`
	TaskCount int    `json:"task_count"`
	CreatedAt string `json:"created_at"`
}

type LabelUpdateResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Color     string `json:"color"`
	TaskCount int    `json:"task_count"`
	UpdatedAt string `json:"updated_at"`
}

type LabelDeleteResponse struct {
	Message           string `json:"message"`
	DeletedLabelID    string `json:"deleted_label_id"`
	AffectedTasksCount int    `json:"affected_tasks_count"`
}

type TaskLabelInfo struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Color string `json:"color"`
}

type TaskLabelListResponse struct {
	TaskID  string          `json:"task_id"`
	TaskRef string          `json:"task_ref"`
	Data    []TaskLabelInfo `json:"data"`
	Total   int             `json:"total"`
}

type AddedLabelInfo struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Color string `json:"color"`
}

type AssignLabelsResponse struct {
	TaskID               string           `json:"task_id"`
	TaskRef              string           `json:"task_ref"`
	Added                []AddedLabelInfo `json:"added"`
	SkippedAlreadyAssigned []string         `json:"skipped_already_assigned"`
	TotalLabelsAfter     int              `json:"total_labels_after"`
}

type RemovedLabelInfo struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Color string `json:"color"`
}

type RemoveLabelsResponse struct {
	TaskID              string             `json:"task_id"`
	TaskRef             string             `json:"task_ref"`
	Removed             []RemovedLabelInfo `json:"removed"`
	SkippedNotAssigned  []string           `json:"skipped_not_assigned"`
	TotalLabelsAfter    int                `json:"total_labels_after"`
}
