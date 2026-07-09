package activitylog

const (
	EventTaskCreated        = "task_created"
	EventSubtaskCreated     = "subtask_created"
	EventTaskUpdated        = "task_updated"
	EventColumnChanged      = "column_changed"
	EventAssigneesAdded     = "assignees_added"
	EventAssigneesRemoved   = "assignees_removed"
	EventTaskDeleted        = "task_deleted"

	EventColumnCreated = "column_created"
	EventColumnUpdated = "column_updated"
	EventColumnDeleted = "column_deleted"

	EventProjectCreated   = "project_created"
	EventProjectUpdated   = "project_updated"
	EventProjectArchived  = "archived"
	EventProjectUnarchived = "unarchived"
	EventProjectDeleted   = "project_deleted"
	EventProjectMemberAdded   = "member_added"
	EventProjectMemberRemoved = "member_removed"
	EventProjectRoleChanged   = "role_changed"

	EventWorkspaceCreated         = "workspace_created"
	EventPlanUpgraded             = "plan_upgraded"
	EventWorkspaceDeleted         = "workspace_deleted"
	EventWorkspaceMemberAdded     = "member_added"
	EventWorkspaceMemberRemoved   = "member_removed"
	EventWorkspaceRoleChanged     = "member_role_changed"
	EventOwnershipTransferred     = "ownership_transferred"

	EventCommentCreated = "comment_created"

	EventMemberJoined = "member_joined" // BR-INV-07: Thành viên join workspace qua invite
)
