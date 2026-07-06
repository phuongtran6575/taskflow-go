package activitylog

func TaskCreated(taskRef, title, priority, columnTitle string, assigneeCount int) map[string]interface{} {
	return map[string]interface{}{
		"event":          EventTaskCreated,
		"task_ref":       taskRef,
		"title":          title,
		"priority":       priority,
		"column_title":   columnTitle,
		"assignee_count": assigneeCount,
	}
}

func SubtaskCreated(taskRef, title, parentTaskRef, parentTitle string) map[string]interface{} {
	return map[string]interface{}{
		"event":           EventSubtaskCreated,
		"task_ref":        taskRef,
		"title":           title,
		"parent_task_ref": parentTaskRef,
		"parent_title":    parentTitle,
	}
}

func TaskUpdated(changes []ChangeField) map[string]interface{} {
	m := map[string]interface{}{
		"event": EventTaskUpdated,
	}
	chgs := make([]map[string]interface{}, len(changes))
	for i, c := range changes {
		mc := map[string]interface{}{
			"field":     c.Field,
			"old_value": c.OldValue,
			"new_value": c.NewValue,
		}
		if c.Display != nil {
			mc["display"] = map[string]string{
				"field_label": c.Display.FieldLabel,
				"old_display": c.Display.OldDisplay,
				"new_display": c.Display.NewDisplay,
			}
		}
		chgs[i] = mc
	}
	m["changes"] = chgs
	return m
}

func ColumnChanged(fromColumnID, fromColumnTitle, toColumnID, toColumnTitle string) map[string]interface{} {
	return map[string]interface{}{
		"event":             EventColumnChanged,
		"from_column_id":    fromColumnID,
		"from_column_title": fromColumnTitle,
		"to_column_id":      toColumnID,
		"to_column_title":   toColumnTitle,
	}
}

func AssigneesAdded(added []map[string]string) map[string]interface{} {
	return map[string]interface{}{
		"event": EventAssigneesAdded,
		"added": added,
	}
}

func AssigneesRemoved(removed []map[string]string) map[string]interface{} {
	return map[string]interface{}{
		"event":   EventAssigneesRemoved,
		"removed": removed,
	}
}

func TaskDeleted(taskRef, title string, deletedSubtasks int) map[string]interface{} {
	return map[string]interface{}{
		"event":                 EventTaskDeleted,
		"task_ref":              taskRef,
		"title":                 title,
		"deleted_subtasks_count": deletedSubtasks,
	}
}

func ColumnCreated(title string, position int) map[string]interface{} {
	return map[string]interface{}{
		"event":    EventColumnCreated,
		"title":    title,
		"position": position,
	}
}

func ColumnUpdated(oldTitle, newTitle string) map[string]interface{} {
	return map[string]interface{}{
		"event":     EventColumnUpdated,
		"old_title": oldTitle,
		"new_title": newTitle,
	}
}

func ColumnDeleted(title string) map[string]interface{} {
	return map[string]interface{}{
		"event": EventColumnDeleted,
		"title": title,
	}
}

func ColumnDeletedMoveTasks(title, targetColumnID, targetColumnTitle string, tasksMoved int) map[string]interface{} {
	return map[string]interface{}{
		"event":              EventColumnDeleted,
		"title":              title,
		"strategy":           "move",
		"target_column_id":   targetColumnID,
		"target_column_title": targetColumnTitle,
		"tasks_moved":        tasksMoved,
	}
}

func ColumnDeletedDeleteTasks(title string, tasksDeleted int) map[string]interface{} {
	return map[string]interface{}{
		"event":         EventColumnDeleted,
		"title":         title,
		"strategy":      "delete",
		"tasks_deleted": tasksDeleted,
	}
}

func ProjectCreated(name, key string) map[string]interface{} {
	return map[string]interface{}{
		"event": EventProjectCreated,
		"name":  name,
		"key":   key,
	}
}

func ProjectUpdated(changes []ChangeField) map[string]interface{} {
	m := map[string]interface{}{
		"event": EventProjectUpdated,
	}
	chgs := make([]map[string]interface{}, len(changes))
	for i, c := range changes {
		mc := map[string]interface{}{
			"field":     c.Field,
			"old_value": c.OldValue,
			"new_value": c.NewValue,
		}
		if c.Display != nil {
			mc["display"] = map[string]string{
				"field_label": c.Display.FieldLabel,
				"old_display": c.Display.OldDisplay,
				"new_display": c.Display.NewDisplay,
			}
		}
		chgs[i] = mc
	}
	m["changes"] = chgs
	return m
}

func ProjectArchived() map[string]interface{} {
	return map[string]interface{}{
		"event": EventProjectArchived,
	}
}

func ProjectUnarchived() map[string]interface{} {
	return map[string]interface{}{
		"event": EventProjectUnarchived,
	}
}

func ProjectDeleted(name, key string, totalTasksDeleted int) map[string]interface{} {
	return map[string]interface{}{
		"event":               EventProjectDeleted,
		"name":                name,
		"key":                 key,
		"total_tasks_deleted": totalTasksDeleted,
	}
}

func ProjectMemberAdded(addedUsers []map[string]string) map[string]interface{} {
	return map[string]interface{}{
		"event":       EventProjectMemberAdded,
		"added_users": addedUsers,
	}
}

func ProjectMemberRemoved(userID, fullName, removedBy string) map[string]interface{} {
	return map[string]interface{}{
		"event":      EventProjectMemberRemoved,
		"user_id":    userID,
		"full_name":  fullName,
		"removed_by": removedBy,
	}
}

func ProjectRoleChanged(userID, fullName, oldRoleName, newRoleName string) map[string]interface{} {
	return map[string]interface{}{
		"event":          EventProjectRoleChanged,
		"user_id":        userID,
		"full_name":      fullName,
		"old_role_name":  oldRoleName,
		"new_role_name":  newRoleName,
	}
}

func WorkspaceCreated(name, plan string) map[string]interface{} {
	return map[string]interface{}{
		"event": EventWorkspaceCreated,
		"name":  name,
		"plan":  plan,
	}
}

func WorkspaceUpdated(changes []ChangeField) map[string]interface{} {
	m := map[string]interface{}{
		"event": "workspace_updated",
	}
	chgs := make([]map[string]interface{}, len(changes))
	for i, c := range changes {
		mc := map[string]interface{}{
			"field":     c.Field,
			"old_value": c.OldValue,
			"new_value": c.NewValue,
		}
		if c.Display != nil {
			mc["display"] = map[string]string{
				"field_label": c.Display.FieldLabel,
				"old_display": c.Display.OldDisplay,
				"new_display": c.Display.NewDisplay,
			}
		}
		chgs[i] = mc
	}
	m["changes"] = chgs
	return m
}

func PlanUpgraded(oldPlan, newPlan string) map[string]interface{} {
	return map[string]interface{}{
		"event":    EventPlanUpgraded,
		"old_plan": oldPlan,
		"new_plan": newPlan,
	}
}

func WorkspaceDeleted(name string) map[string]interface{} {
	return map[string]interface{}{
		"event": EventWorkspaceDeleted,
		"name":  name,
	}
}

func WorkspaceMemberAdded(userID, fullName, role string) map[string]interface{} {
	return map[string]interface{}{
		"event":     EventWorkspaceMemberAdded,
		"user_id":   userID,
		"full_name": fullName,
		"role":      role,
	}
}

func WorkspaceMemberRemoved(userID, fullName, removedBy string) map[string]interface{} {
	return map[string]interface{}{
		"event":      EventWorkspaceMemberRemoved,
		"user_id":    userID,
		"full_name":  fullName,
		"removed_by": removedBy,
	}
}

func WorkspaceRoleChanged(userID, fullName, oldRole, newRole string) map[string]interface{} {
	return map[string]interface{}{
		"event":    EventWorkspaceRoleChanged,
		"user_id":  userID,
		"full_name": fullName,
		"old_role": oldRole,
		"new_role": newRole,
	}
}

func OwnershipTransferred(fromUserID, fromFullName, toUserID, toFullName string) map[string]interface{} {
	return map[string]interface{}{
		"event":         EventOwnershipTransferred,
		"from_user_id":  fromUserID,
		"from_full_name": fromFullName,
		"to_user_id":    toUserID,
		"to_full_name":  toFullName,
	}
}

func CommentCreated(commentID, contentPreview string, mentionCount int) map[string]interface{} {
	return map[string]interface{}{
		"event":          EventCommentCreated,
		"comment_id":     commentID,
		"content_preview": contentPreview,
		"mention_count":  mentionCount,
	}
}
