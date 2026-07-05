package _interface

import (
	"TaskFlow-Go/internal/dto"
	"TaskFlow-Go/internal/models"
)

type CommentRepository interface {
	Create(comment *models.Comment) error
	GetByID(id string) (*models.Comment, error)
	Update(comment *models.Comment) error
	Delete(id string) error

	ListByTaskIDWithCursor(taskID string, limit int, cursor string, direction string) (*dto.CommentListResponse, error)
	ListMentionableUsers(projectID string, search string, limit int) (*dto.MentionableUsersResponse, error)

	CreateMention(mention *models.CommentMention) error
	DeleteMentionsByCommentID(commentID string) error
	GetMentionsByCommentID(commentID string) ([]models.CommentMention, error)
	ResolveUsernames(projectID string, workspaceID string, usernames []string) (map[string]dto.MentionUser, error)

	ListPreviousCommenters(taskID string, excludeUserID string) ([]string, error)
}
