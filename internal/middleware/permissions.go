package middleware

// Permission slugs cho hệ thống RBAC.
// Format: "<module>.<action>"
// Các slug này phải khớp với cột `slug` trong bảng `permissions`.
//
// Khi thêm permission mới:
//  1. Thêm constant ở đây
//  2. Seed vào DB (bảng permissions)
//  3. Gán cho role phù hợp (bảng role_permissions)

// --- Task permissions ---
const (
	PermTaskView   = "task.view"
	PermTaskCreate = "task.create"
	PermTaskEdit   = "task.edit"
	PermTaskDelete = "task.delete"
	PermTaskAssign = "task.assign"
	PermTaskMove   = "task.move"
)

// --- Column permissions ---
const (
	PermColumnView   = "column.view"
	PermColumnCreate = "column.create"
	PermColumnEdit   = "column.edit"
	PermColumnDelete = "column.delete"
)

// --- Label permissions ---
const (
	PermLabelView   = "label.view"
	PermLabelCreate = "label.create"
	PermLabelEdit   = "label.edit"
	PermLabelDelete = "label.delete"
)

// --- Attachment permissions ---
const (
	PermAttachmentView   = "attachment.view"
	PermAttachmentUpload = "attachment.upload"
	PermAttachmentDelete = "attachment.delete"
)

// --- Comment permissions ---
const (
	PermCommentView   = "comment.view"
	PermCommentCreate = "comment.create"
	PermCommentEdit   = "comment.edit"
	PermCommentDelete = "comment.delete"
)

// --- Project permissions ---
const (
	PermProjectUpdate  = "project.update"
	PermProjectArchive = "project.archive"
	PermProjectDelete  = "project.delete"
)

// --- Project member permissions ---
const (
	PermMemberView   = "member.view"
	PermMemberAdd    = "member.add"
	PermMemberEdit   = "member.edit"
	PermMemberRemove = "member.remove"
)
