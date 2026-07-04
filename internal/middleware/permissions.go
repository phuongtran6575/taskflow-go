package middleware

// Permission slugs cho hệ thống RBAC.
// Format: "{module}:{action}" (xem BR-PERM-02)
// Các slug này phải khớp với cột `slug` trong bảng `permissions`.
//
// Khi thêm permission mới:
//  1. Thêm constant ở đây
//  2. Seed vào DB (bảng permissions)
//  3. Gán cho role phù hợp (bảng role_permissions)

// --- Task permissions ---
const (
	PermTaskView        = "task:view"
	PermTaskCreate      = "task:create"
	PermTaskUpdate      = "task:update"
	PermTaskDelete      = "task:delete"
	PermTaskAssign      = "task:assign"
	PermTaskMove        = "task:move"
	PermTaskSetPriority = "task:set_priority"
)

// --- Project permissions ---
const (
	PermProjectView          = "project:view"
	PermProjectUpdate        = "project:update"
	PermProjectDelete        = "project:delete"
	PermProjectManageMembers = "project:manage_members"
	PermProjectArchive       = "project:archive"
)

// --- Column permissions ---
const (
	PermColumnCreate = "column:create"
	PermColumnUpdate = "column:update"
	PermColumnDelete = "column:delete"
)

// --- Comment permissions ---
const (
	PermCommentCreate     = "comment:create"
	PermCommentUpdateOwn  = "comment:update_own"
	PermCommentDeleteOwn  = "comment:delete_own"
	PermCommentDeleteAny  = "comment:delete_any"
)

// --- Label permissions ---
const (
	PermLabelCreate = "label:create"
	PermLabelUpdate = "label:update"
	PermLabelDelete = "label:delete"
	PermLabelAssign = "label:assign"
)

// --- Attachment permissions ---
const (
	PermAttachmentUpload    = "attachment:upload"
	PermAttachmentDeleteOwn = "attachment:delete_own"
	PermAttachmentDeleteAny = "attachment:delete_any"
)
