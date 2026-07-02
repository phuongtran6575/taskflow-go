package _interface

import "TaskFlow-Go/internal/dto"

type CommentService interface {
	ListComments(workspaceID string, userID string, projectID string, taskID string, limit int, cursor string, direction string) (*dto.CommentListResponse, error)
	CreateComment(workspaceID string, userID string, projectID string, taskID string, req *dto.CreateCommentRequest) (*dto.CommentCreateResponse, error)
	UpdateComment(workspaceID string, userID string, projectID string, taskID string, commentID string, req *dto.UpdateCommentRequest) (*dto.CommentUpdateResponse, error)
	DeleteComment(workspaceID string, userID string, projectID string, taskID string, commentID string) (*dto.CommentDeleteResponse, error)
	GetMentionableUsers(workspaceID string, userID string, projectID string, taskID string, search string, limit int) (*dto.MentionableUsersResponse, error)
}
