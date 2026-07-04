package apperror

import "net/http"

type AppError struct {
	Status  int    `json:"-"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *AppError) Error() string {
	return e.Message
}

var (
	// Auth errors
	ErrInvalidCredentials  = &AppError{Status: http.StatusUnauthorized, Code: "INVALID_CREDENTIALS", Message: "Invalid email or password"}
	ErrAccountDisabled     = &AppError{Status: http.StatusForbidden, Code: "ACCOUNT_DISABLED", Message: "Account has been disabled"}
	ErrWrongAuthProvider   = &AppError{Status: http.StatusForbidden, Code: "WRONG_AUTH_PROVIDER", Message: "This email uses a different authentication provider"}
	ErrInvalidRefreshToken = &AppError{Status: http.StatusUnauthorized, Code: "INVALID_REFRESH_TOKEN", Message: "Invalid refresh token"}
	ErrRefreshTokenExpired = &AppError{Status: http.StatusUnauthorized, Code: "REFRESH_TOKEN_EXPIRED", Message: "Refresh token has expired"}
	ErrRefreshTokenReuse   = &AppError{Status: http.StatusUnauthorized, Code: "REFRESH_TOKEN_REUSE", Message: "Refresh token has been revoked"}
	ErrInvalidResetToken   = &AppError{Status: http.StatusBadRequest, Code: "INVALID_RESET_TOKEN", Message: "Invalid or already used reset token"}
	ErrResetTokenExpired   = &AppError{Status: http.StatusBadRequest, Code: "RESET_TOKEN_EXPIRED", Message: "Reset token has expired (15 minutes)"}
	ErrSameAsOldPassword   = &AppError{Status: http.StatusBadRequest, Code: "SAME_AS_OLD_PASSWORD", Message: "New password must be different from current password"}
	ErrPasswordMismatch    = &AppError{Status: http.StatusBadRequest, Code: "PASSWORD_MISMATCH", Message: "Passwords do not match"}
	ErrWeakPassword        = &AppError{Status: http.StatusBadRequest, Code: "WEAK_PASSWORD", Message: "Password does not meet complexity requirements"}
	ErrEmailAlreadyExists  = &AppError{Status: http.StatusConflict, Code: "EMAIL_ALREADY_EXISTS", Message: "Email is already registered"}
	ErrValidation          = &AppError{Status: http.StatusBadRequest, Code: "VALIDATION_ERROR", Message: "Validation failed"}
	ErrUnauthorized        = &AppError{Status: http.StatusUnauthorized, Code: "UNAUTHORIZED", Message: "Authentication required"}

	// User errors
	ErrUsernameAlreadyTaken       = &AppError{Status: http.StatusConflict, Code: "USERNAME_ALREADY_TAKEN", Message: "Username is already taken"}
	ErrWrongCurrentPassword       = &AppError{Status: http.StatusUnauthorized, Code: "WRONG_CURRENT_PASSWORD", Message: "Current password is incorrect"}
	ErrOAuthAccountNoPassword     = &AppError{Status: http.StatusBadRequest, Code: "OAUTH_ACCOUNT_NO_PASSWORD", Message: "OAuth account has no password set"}
	ErrInvalidFileType            = &AppError{Status: http.StatusBadRequest, Code: "INVALID_FILE_TYPE", Message: "File type not supported (JPG/PNG/WEBP only)"}
	ErrFileTooLarge               = &AppError{Status: http.StatusBadRequest, Code: "FILE_TOO_LARGE", Message: "File exceeds maximum size"}
	ErrCannotRevokeCurrentSession = &AppError{Status: http.StatusBadRequest, Code: "CANNOT_REVOKE_CURRENT_SESSION", Message: "Use /logout to revoke current session"}
	ErrSessionNotFound            = &AppError{Status: http.StatusNotFound, Code: "SESSION_NOT_FOUND", Message: "Session not found"}
	ErrInvalidConfirmation        = &AppError{Status: http.StatusBadRequest, Code: "INVALID_CONFIRMATION", Message: "Confirmation text does not match"}
	ErrWorkspaceOwnerConflict     = &AppError{Status: http.StatusConflict, Code: "WORKSPACE_OWNER_CONFLICT", Message: "You are still an OWNER of one or more workspaces"}

	// Workspace errors
	ErrDomainAlreadyTaken      = &AppError{Status: http.StatusConflict, Code: "DOMAIN_ALREADY_TAKEN", Message: "Domain is already in use"}
	ErrWorkspaceLimitReached   = &AppError{Status: http.StatusTooManyRequests, Code: "WORKSPACE_LIMIT_REACHED", Message: "Workspace limit reached for your plan"}
	ErrWorkspaceNotFound       = &AppError{Status: http.StatusNotFound, Code: "WORKSPACE_NOT_FOUND", Message: "Workspace not found"}
	ErrNotAMember              = &AppError{Status: http.StatusForbidden, Code: "NOT_A_MEMBER", Message: "You are not a member of this workspace"}
	ErrForbidden               = &AppError{Status: http.StatusForbidden, Code: "FORBIDDEN", Message: "You do not have permission"}
	ErrInvalidPlan             = &AppError{Status: http.StatusBadRequest, Code: "INVALID_PLAN", Message: "Invalid plan value"}
	ErrPlanDowngradeNotAllowed = &AppError{Status: http.StatusBadRequest, Code: "PLAN_DOWNGRADE_NOT_ALLOWED", Message: "Plan downgrade is not allowed"}
	ErrAlreadyOnThisPlan       = &AppError{Status: http.StatusBadRequest, Code: "ALREADY_ON_THIS_PLAN", Message: "Workspace is already on this plan"}

	// Member errors
	ErrMemberNotFound        = &AppError{Status: http.StatusNotFound, Code: "MEMBER_NOT_FOUND", Message: "Member not found"}
	ErrInvalidRole           = &AppError{Status: http.StatusBadRequest, Code: "INVALID_ROLE", Message: "Invalid role value"}
	ErrSameRole              = &AppError{Status: http.StatusBadRequest, Code: "SAME_ROLE", Message: "Role is unchanged"}
	ErrCannotChangeOwnRole   = &AppError{Status: http.StatusBadRequest, Code: "CANNOT_CHANGE_OWN_ROLE", Message: "You cannot change your own role"}
	ErrCannotAssignOwnerRole = &AppError{Status: http.StatusBadRequest, Code: "CANNOT_ASSIGN_OWNER_ROLE", Message: "Use transfer-ownership endpoint instead"}
	ErrCannotTransferToSelf  = &AppError{Status: http.StatusBadRequest, Code: "CANNOT_TRANSFER_TO_SELF", Message: "Cannot transfer ownership to yourself"}
	ErrTargetAccountDisabled = &AppError{Status: http.StatusUnprocessableEntity, Code: "TARGET_ACCOUNT_DISABLED", Message: "Target account is disabled"}
	ErrCannotKickSelf        = &AppError{Status: http.StatusBadRequest, Code: "CANNOT_KICK_SELF", Message: "Use DELETE /me instead"}
	ErrCannotKickOwner       = &AppError{Status: http.StatusForbidden, Code: "CANNOT_KICK_OWNER", Message: "Cannot kick the workspace OWNER"}
	ErrConfirmationRequired  = &AppError{Status: http.StatusBadRequest, Code: "CONFIRMATION_REQUIRED", Message: "Confirmation is required"}
	ErrOwnerCannotLeave      = &AppError{Status: http.StatusUnprocessableEntity, Code: "OWNER_CANNOT_LEAVE", Message: "Transfer ownership before leaving"}

	// Invite errors
	ErrInviteNotFound         = &AppError{Status: http.StatusNotFound, Code: "INVITE_NOT_FOUND", Message: "Invite code not found"}
	ErrInviteExpired          = &AppError{Status: http.StatusGone, Code: "INVITE_EXPIRED", Message: "Invite link has expired"}
	ErrInviteExhausted        = &AppError{Status: http.StatusGone, Code: "INVITE_EXHAUSTED", Message: "Invite link has reached maximum uses"}
	ErrInviteRevoked          = &AppError{Status: http.StatusGone, Code: "INVITE_REVOKED", Message: "Invite link has been revoked"}
	ErrAlreadyAMember         = &AppError{Status: http.StatusConflict, Code: "ALREADY_A_MEMBER", Message: "You are already a member of this workspace"}
	ErrInviteLinkLimitReached = &AppError{Status: http.StatusTooManyRequests, Code: "INVITE_LINK_LIMIT_REACHED", Message: "Invite link limit reached"}
	ErrAlreadyRevoked         = &AppError{Status: http.StatusConflict, Code: "ALREADY_REVOKED", Message: "Invite link has already been revoked"}

	// RBAC errors
	ErrPermissionNotFound    = &AppError{Status: http.StatusNotFound, Code: "PERMISSION_NOT_FOUND", Message: "Permission not found"}
	ErrRoleNotFound          = &AppError{Status: http.StatusNotFound, Code: "ROLE_NOT_FOUND", Message: "Role not found"}
	ErrRoleNameAlreadyExists = &AppError{Status: http.StatusConflict, Code: "ROLE_NAME_ALREADY_EXISTS", Message: "Role name already exists in this workspace"}
	ErrRoleLimitReached      = &AppError{Status: http.StatusTooManyRequests, Code: "ROLE_LIMIT_REACHED", Message: "Role limit reached for your plan"}
	ErrPermissionIDsRequired = &AppError{Status: http.StatusBadRequest, Code: "PERMISSION_IDS_REQUIRED", Message: "Permission IDs are required"}
	ErrInvalidPermissionIDs  = &AppError{Status: http.StatusBadRequest, Code: "INVALID_PERMISSION_IDS", Message: "One or more permission IDs are invalid"}
	ErrRoleInUse             = &AppError{Status: http.StatusConflict, Code: "ROLE_IN_USE", Message: "Role is currently assigned to project members"}

	// Project member errors
	ErrMembersRequired               = &AppError{Status: http.StatusBadRequest, Code: "MEMBERS_REQUIRED", Message: "Members array is required"}
	ErrInvalidRoleID                 = &AppError{Status: http.StatusBadRequest, Code: "INVALID_ROLE_ID", Message: "One or more role IDs are invalid"}
	ErrUserNotInWorkspace            = &AppError{Status: http.StatusBadRequest, Code: "USER_NOT_IN_WORKSPACE", Message: "One or more users are not workspace members"}
	ErrCannotRemoveSelf              = &AppError{Status: http.StatusBadRequest, Code: "CANNOT_REMOVE_SELF", Message: "Use DELETE /me instead"}
	ErrCannotRemoveWorkspaceOwner    = &AppError{Status: http.StatusBadRequest, Code: "CANNOT_REMOVE_WORKSPACE_OWNER", Message: "Cannot remove workspace OWNER from project"}
	ErrCannotChangeWorkspaceOwnerRole = &AppError{Status: http.StatusBadRequest, Code: "CANNOT_CHANGE_WORKSPACE_OWNER_ROLE", Message: "Cannot change role of workspace OWNER"}
	ErrWorkspaceOwnerCannotLeave     = &AppError{Status: http.StatusUnprocessableEntity, Code: "WORKSPACE_OWNER_CANNOT_LEAVE", Message: "Workspace owner cannot leave the project"}

	// Project errors
	ErrProjectNotFound         = &AppError{Status: http.StatusNotFound, Code: "PROJECT_NOT_FOUND", Message: "Project not found"}
	ErrProjectKeyAlreadyExists = &AppError{Status: http.StatusConflict, Code: "PROJECT_KEY_ALREADY_EXISTS", Message: "Project key already exists in this workspace"}
	ErrProjectLimitReached     = &AppError{Status: http.StatusTooManyRequests, Code: "PROJECT_LIMIT_REACHED", Message: "Project limit reached for your plan"}
	ErrNotAProjectMember       = &AppError{Status: http.StatusForbidden, Code: "NOT_A_PROJECT_MEMBER", Message: "You are not a member of this project"}
	ErrProjectArchived         = &AppError{Status: http.StatusForbidden, Code: "PROJECT_ARCHIVED", Message: "Project is archived"}
	ErrAlreadyArchived         = &AppError{Status: http.StatusBadRequest, Code: "ALREADY_ARCHIVED", Message: "Project is already archived"}
	ErrNotArchived             = &AppError{Status: http.StatusBadRequest, Code: "NOT_ARCHIVED", Message: "Project is not archived"}
	ErrKeyIsImmutable          = &AppError{Status: http.StatusBadRequest, Code: "KEY_IS_IMMUTABLE", Message: "Project key cannot be changed"}
	ErrInvalidKeyFormat        = &AppError{Status: http.StatusBadRequest, Code: "INVALID_KEY_FORMAT", Message: "Invalid project key format"}
	ErrInvalidBackground       = &AppError{Status: http.StatusBadRequest, Code: "INVALID_BACKGROUND", Message: "Background must be a valid HEX color or URL"}

	// Column errors
	ErrColumnNotFound         = &AppError{Status: http.StatusNotFound, Code: "COLUMN_NOT_FOUND", Message: "Column not found"}
	ErrColumnLimitReached     = &AppError{Status: http.StatusTooManyRequests, Code: "COLUMN_LIMIT_REACHED", Message: "Maximum 20 columns per project"}
	ErrInvalidPositionContext = &AppError{Status: http.StatusBadRequest, Code: "INVALID_POSITION_CONTEXT", Message: "Invalid position context"}
	ErrStrategyRequired       = &AppError{Status: http.StatusBadRequest, Code: "STRATEGY_REQUIRED", Message: "Delete strategy is required when column has tasks"}
	ErrTargetColumnRequired   = &AppError{Status: http.StatusBadRequest, Code: "TARGET_COLUMN_REQUIRED", Message: "Target column is required for move strategy"}
	ErrInvalidTargetColumn    = &AppError{Status: http.StatusBadRequest, Code: "INVALID_TARGET_COLUMN", Message: "Invalid target column"}
	ErrCannotMoveToSelf       = &AppError{Status: http.StatusBadRequest, Code: "CANNOT_MOVE_TO_SELF", Message: "Cannot move tasks to the same column being deleted"}
	ErrLastColumn             = &AppError{Status: http.StatusBadRequest, Code: "LAST_COLUMN", Message: "Cannot delete the last column"}

	// Task errors
	ErrTaskNotFound                 = &AppError{Status: http.StatusNotFound, Code: "TASK_NOT_FOUND", Message: "Task not found"}
	ErrInvalidColumn                = &AppError{Status: http.StatusBadRequest, Code: "INVALID_COLUMN", Message: "Column does not belong to this project"}
	ErrInvalidDateRange             = &AppError{Status: http.StatusBadRequest, Code: "INVALID_DATE_RANGE", Message: "Start date must be before due date"}
	ErrInvalidAssigneeIDs           = &AppError{Status: http.StatusBadRequest, Code: "INVALID_ASSIGNEE_IDS", Message: "One or more assignees are not project members"}
	ErrInvalidLabelIDs              = &AppError{Status: http.StatusBadRequest, Code: "INVALID_LABEL_IDS", Message: "One or more labels do not belong to this project"}
	ErrInvalidPriority              = &AppError{Status: http.StatusBadRequest, Code: "INVALID_PRIORITY", Message: "Invalid priority value"}
	ErrCannotCreateSubtaskOfSubtask = &AppError{Status: http.StatusBadRequest, Code: "CANNOT_CREATE_SUBTASK_OF_SUBTASK", Message: "Cannot create a subtask of another subtask"}
	ErrSubtaskLimitReached          = &AppError{Status: http.StatusBadRequest, Code: "SUBTASK_LIMIT_REACHED", Message: "Maximum 100 subtasks per task"}
	ErrCannotMoveDeletedTask        = &AppError{Status: http.StatusBadRequest, Code: "CANNOT_MOVE_DELETED_TASK", Message: "Cannot move a deleted task"}
	ErrPositionConflict             = &AppError{Status: http.StatusConflict, Code: "POSITION_CONFLICT", Message: "Position changed by another user, please refetch"}

	// Assignee errors
	ErrUserIDsRequired      = &AppError{Status: http.StatusBadRequest, Code: "USER_IDS_REQUIRED", Message: "User IDs are required"}
	ErrAssigneeLimitReached = &AppError{Status: http.StatusBadRequest, Code: "ASSIGNEE_LIMIT_REACHED", Message: "Maximum 20 assignees per task"}
	ErrInvalidUserIDs       = &AppError{Status: http.StatusBadRequest, Code: "INVALID_USER_IDS", Message: "One or more users are not project members"}
	ErrAlreadyAssigned      = &AppError{Status: http.StatusBadRequest, Code: "ALREADY_ASSIGNED", Message: "User is already assigned to this task"}

	// Label errors
	ErrLabelNotFound          = &AppError{Status: http.StatusNotFound, Code: "LABEL_NOT_FOUND", Message: "Label not found"}
	ErrInvalidColor           = &AppError{Status: http.StatusBadRequest, Code: "INVALID_COLOR", Message: "Invalid HEX color"}
	ErrLabelNameAlreadyExists = &AppError{Status: http.StatusConflict, Code: "LABEL_NAME_ALREADY_EXISTS", Message: "Label name already exists in this project"}
	ErrLabelLimitReached      = &AppError{Status: http.StatusTooManyRequests, Code: "LABEL_LIMIT_REACHED", Message: "Maximum 50 labels per project"}
	ErrLabelIDsRequired       = &AppError{Status: http.StatusBadRequest, Code: "LABEL_IDS_REQUIRED", Message: "Label IDs are required"}
	ErrTaskLabelLimitReached  = &AppError{Status: http.StatusBadRequest, Code: "LABEL_LIMIT_REACHED", Message: "Maximum 10 labels per task"}

	// Attachment errors
	ErrNoFilesProvided      = &AppError{Status: http.StatusBadRequest, Code: "NO_FILES_PROVIDED", Message: "No files provided"}
	ErrBatchSizeExceeded    = &AppError{Status: http.StatusBadRequest, Code: "BATCH_SIZE_EXCEEDED", Message: "Maximum 10 files per request"}
	ErrStorageQuotaExceeded = &AppError{Status: http.StatusInsufficientStorage, Code: "STORAGE_QUOTA_EXCEEDED", Message: "Workspace storage quota exceeded"}
	ErrAttachmentNotFound   = &AppError{Status: http.StatusNotFound, Code: "ATTACHMENT_NOT_FOUND", Message: "Attachment not found"}
	ErrStorageUnavailable   = &AppError{Status: http.StatusServiceUnavailable, Code: "STORAGE_UNAVAILABLE", Message: "Storage service unavailable"}

	// Comment errors
	ErrContentRequired = &AppError{Status: http.StatusBadRequest, Code: "CONTENT_REQUIRED", Message: "Content is required"}
	ErrContentTooLong  = &AppError{Status: http.StatusBadRequest, Code: "CONTENT_TOO_LONG", Message: "Content exceeds 10,000 characters"}
	ErrCommentNotFound = &AppError{Status: http.StatusNotFound, Code: "COMMENT_NOT_FOUND", Message: "Comment not found"}

	// Notification errors
	ErrNotificationNotFound = &AppError{Status: http.StatusNotFound, Code: "NOTIFICATION_NOT_FOUND", Message: "Notification not found"}
	ErrInvalidTargetRoles   = &AppError{Status: http.StatusBadRequest, Code: "INVALID_TARGET_ROLES", Message: "Invalid target roles"}

	// Member limit
	ErrWorkspaceMemberLimitReached = &AppError{Status: http.StatusTooManyRequests, Code: "WORKSPACE_MEMBER_LIMIT_REACHED", Message: "Workspace member limit reached"}
	ErrCustomRolesNotAllowed      = &AppError{Status: http.StatusForbidden, Code: "CUSTOM_ROLES_NOT_ALLOWED", Message: "Custom roles are not available on your current plan"}

	// Upload errors
	ErrUploadFailed = &AppError{Status: http.StatusInternalServerError, Code: "UPLOAD_FAILED", Message: "File upload failed"}
)

func NewAppError(status int, code, message string) *AppError {
	return &AppError{Status: status, Code: code, Message: message}
}
