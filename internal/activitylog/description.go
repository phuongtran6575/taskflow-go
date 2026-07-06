package activitylog

import (
	"fmt"
	"strings"
)

func GenerateDescription(actorName string, metadata map[string]interface{}) string {
	if metadata == nil {
		return ""
	}
	event, _ := metadata["event"].(string)
	actor := actorName
	if actor == "" {
		actor = "Hệ thống"
	}

	switch event {
	case EventTaskCreated:
		taskRef, _ := metadata["task_ref"].(string)
		title, _ := metadata["title"].(string)
		title = Truncate(title, 50)
		return fmt.Sprintf("%s đã tạo task %s: %s", actor, taskRef, title)

	case EventSubtaskCreated:
		taskRef, _ := metadata["task_ref"].(string)
		title, _ := metadata["title"].(string)
		parentRef, _ := metadata["parent_task_ref"].(string)
		title = Truncate(title, 50)
		return fmt.Sprintf("%s đã tạo subtask %s: %s trong %s", actor, taskRef, title, parentRef)

	case EventTaskUpdated:
		taskRef, _ := metadata["task_ref"].(string)
		title, _ := metadata["title"].(string)
		changes, _ := metadata["changes"].([]interface{})
		if len(changes) == 1 {
			c := changes[0].(map[string]interface{})
			display, ok := c["display"].(map[string]interface{})
			if ok {
				fieldLabel, _ := display["field_label"].(string)
				oldDisp, _ := display["old_display"].(string)
				newDisp, _ := display["new_display"].(string)
				return fmt.Sprintf("%s đã thay đổi %s của %s từ '%s' sang '%s'", actor, fieldLabel, taskRef, oldDisp, newDisp)
			}
		}
		if title != "" {
			return fmt.Sprintf("%s đã cập nhật %s: %s (%d thay đổi)", actor, taskRef, Truncate(title, 50), len(changes))
		}
		return fmt.Sprintf("%s đã cập nhật %s (%d thay đổi)", actor, taskRef, len(changes))

	case EventColumnChanged:
		taskRef, _ := metadata["task_ref"].(string)
		fromTitle, _ := metadata["from_column_title"].(string)
		toTitle, _ := metadata["to_column_title"].(string)
		return fmt.Sprintf("%s đã chuyển %s từ '%s' sang '%s'", actor, taskRef, fromTitle, toTitle)

	case EventAssigneesAdded:
		taskRef, _ := metadata["task_ref"].(string)
		added, _ := metadata["added"].([]interface{})
		if len(added) == 1 {
			u := added[0].(map[string]interface{})
			name, _ := u["full_name"].(string)
			return fmt.Sprintf("%s đã giao %s cho %s", actor, taskRef, name)
		}
		return fmt.Sprintf("%s đã giao %s cho %d thành viên", actor, taskRef, len(added))

	case EventAssigneesRemoved:
		taskRef, _ := metadata["task_ref"].(string)
		removed, _ := metadata["removed"].([]interface{})
		if len(removed) == 1 {
			u := removed[0].(map[string]interface{})
			name, _ := u["full_name"].(string)
			return fmt.Sprintf("%s đã bỏ gán %s khỏi %s", actor, name, taskRef)
		}
		return fmt.Sprintf("%s đã bỏ gán %d thành viên khỏi %s", actor, len(removed), taskRef)

	case EventTaskDeleted:
		taskRef, _ := metadata["task_ref"].(string)
		title, _ := metadata["title"].(string)
		return fmt.Sprintf("%s đã xóa task %s: %s", actor, taskRef, Truncate(title, 50))

	case EventColumnCreated:
		title, _ := metadata["title"].(string)
		return fmt.Sprintf("%s đã tạo cột '%s'", actor, title)

	case EventColumnUpdated:
		oldTitle, _ := metadata["old_title"].(string)
		newTitle, _ := metadata["new_title"].(string)
		return fmt.Sprintf("%s đã đổi tên cột từ '%s' sang '%s'", actor, oldTitle, newTitle)

	case EventColumnDeleted:
		title, _ := metadata["title"].(string)
		strategy, _ := metadata["strategy"].(string)
		if strategy == "move" {
			targetCol, _ := metadata["target_column_title"].(string)
			tasksMoved, _ := metadata["tasks_moved"].(float64)
			return fmt.Sprintf("%s đã xóa cột '%s' và chuyển %d task sang '%s'", actor, title, int(tasksMoved), targetCol)
		}
		if strategy == "delete" {
			tasksDel, _ := metadata["tasks_deleted"].(float64)
			return fmt.Sprintf("%s đã xóa cột '%s' và %d task trong đó", actor, title, int(tasksDel))
		}
		return fmt.Sprintf("%s đã xóa cột '%s'", actor, title)

	case EventProjectCreated:
		name, _ := metadata["name"].(string)
		key, _ := metadata["key"].(string)
		return fmt.Sprintf("%s đã tạo project %s: %s", actor, key, name)

	case EventProjectArchived:
		return fmt.Sprintf("%s đã archive project", actor)

	case EventProjectUnarchived:
		return fmt.Sprintf("%s đã unarchive project", actor)

	case EventProjectDeleted:
		name, _ := metadata["name"].(string)
		key, _ := metadata["key"].(string)
		return fmt.Sprintf("%s đã xóa project %s: %s", actor, key, name)

	case EventProjectMemberAdded:
		addedUsers, ok := metadata["added_users"].([]interface{})
		if ok && len(addedUsers) > 0 {
			names := make([]string, len(addedUsers))
			for i, u := range addedUsers {
				um := u.(map[string]interface{})
				names[i], _ = um["full_name"].(string)
			}
			return fmt.Sprintf("%s đã thêm %s vào project", actor, strings.Join(names, ", "))
		}
		fullName, _ := metadata["full_name"].(string)
		role, _ := metadata["role"].(string)
		if role != "" {
			return fmt.Sprintf("%s đã thêm %s vào workspace với vai trò %s", actor, fullName, role)
		}
		return fmt.Sprintf("%s đã thêm %s vào workspace", actor, fullName)

	case EventProjectMemberRemoved:
		fullName, _ := metadata["full_name"].(string)
		return fmt.Sprintf("%s đã xóa %s", actor, fullName)

	case EventWorkspaceRoleChanged:
		fullName, _ := metadata["full_name"].(string)
		oldRole, _ := metadata["old_role"].(string)
		newRole, _ := metadata["new_role"].(string)
		return fmt.Sprintf("%s đã đổi role của %s từ %s sang %s", actor, fullName, oldRole, newRole)

	case EventProjectRoleChanged:
		fullName, _ := metadata["full_name"].(string)
		oldRole, _ := metadata["old_role_name"].(string)
		newRole, _ := metadata["new_role_name"].(string)
		return fmt.Sprintf("%s đã đổi role của %s từ %s sang %s", actor, fullName, oldRole, newRole)

	case EventPlanUpgraded:
		newPlan, _ := metadata["new_plan"].(string)
		return fmt.Sprintf("%s đã nâng cấp workspace lên plan %s", actor, newPlan)

	case EventOwnershipTransferred:
		toName, _ := metadata["to_full_name"].(string)
		return fmt.Sprintf("%s đã chuyển quyền OWNER cho %s", actor, toName)

	case EventWorkspaceCreated:
		name, _ := metadata["name"].(string)
		return fmt.Sprintf("%s đã tạo workspace %s", actor, name)

	case EventWorkspaceDeleted:
		name, _ := metadata["name"].(string)
		return fmt.Sprintf("%s đã xóa workspace %s", actor, name)

	case EventCommentCreated:
		preview, _ := metadata["content_preview"].(string)
		mCount, _ := metadata["mention_count"].(float64)
		desc := fmt.Sprintf("%s đã bình luận: \"%s\"", actor, preview)
		if int(mCount) > 0 {
			desc += fmt.Sprintf(" (đề cập %d người)", int(mCount))
		}
		return desc

	default:
		if action, ok := metadata["action"].(string); ok {
			return fmt.Sprintf("%s đã thực hiện %s", actor, strings.ToLower(action))
		}
		return ""
	}
}
