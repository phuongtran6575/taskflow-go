package helper

import (
	"TaskFlow-Go/internal/dto"
	"TaskFlow-Go/internal/models"
)

func ColumnsToInfo(cols []models.Column) []dto.ColumnInfo {
	res := make([]dto.ColumnInfo, len(cols))
	for i, c := range cols {
		res[i] = dto.ColumnInfo{
			ID:        c.ID,
			Title:     c.Title,
			Position:  c.Position,
			IsDone:    c.IsDone,
			TaskCount: 0,
			CreatedAt: c.CreatedAt,
			UpdatedAt: c.UpdatedAt,
		}
	}
	return res
}

func ToActivityEntry(item dto.ActivityLogInfo) dto.TimelineActivityEntry {
	return dto.TimelineActivityEntry{
		EntryType:   "activity",
		ID:          item.ID,
		Action:      item.Action,
		EntityType:  item.EntityType,
		Description: item.Description,
		Actor:       item.Actor,
		Metadata:    item.Metadata,
		CreatedAt:   item.CreatedAt,
	}
}

func ToCommentEntry(item dto.CommentInfo) dto.TimelineCommentEntry {
	return dto.TimelineCommentEntry{
		EntryType:   "comment",
		ID:          item.ID,
		Content:     item.Content,
		ContentHTML: item.ContentHTML,
		IsDeleted:   item.IsDeleted,
		IsEdited:    item.IsEdited,
		Author: &dto.ActivityLogActor{
			UserID:    item.Author.UserID,
			FullName:  item.Author.FullName,
			Username:  item.Author.Username,
			AvatarURL: item.Author.AvatarURL,
		},
		Mentions:  item.Mentions,
		CreatedAt: item.CreatedAt,
		UpdatedAt: item.UpdatedAt,
	}
}

func MergeTimeline(activities []dto.ActivityLogInfo, comments []dto.CommentInfo, direction string) []interface{} {
	result := make([]interface{}, 0, len(activities)+len(comments))
	aIdx, cIdx := 0, 0
	asc := direction == "asc"

	for aIdx < len(activities) && cIdx < len(comments) {
		var before bool
		if asc {
			before = activities[aIdx].CreatedAt <= comments[cIdx].CreatedAt
		} else {
			before = activities[aIdx].CreatedAt >= comments[cIdx].CreatedAt
		}
		if before {
			result = append(result, ToActivityEntry(activities[aIdx]))
			aIdx++
		} else {
			result = append(result, ToCommentEntry(comments[cIdx]))
			cIdx++
		}
	}
	for aIdx < len(activities) {
		result = append(result, ToActivityEntry(activities[aIdx]))
		aIdx++
	}
	for cIdx < len(comments) {
		result = append(result, ToCommentEntry(comments[cIdx]))
		cIdx++
	}
	return result
}
