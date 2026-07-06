package activitylog

import "time"

type DisplayInfo struct {
	FieldLabel string `json:"field_label"`
	OldDisplay string `json:"old_display"`
	NewDisplay string `json:"new_display"`
}

type ChangeField struct {
	Field    string      `json:"field"`
	OldValue interface{} `json:"old_value"`
	NewValue interface{} `json:"new_value"`
	Display  *DisplayInfo `json:"display,omitempty"`
}

var priorityLabels = map[string]string{
	"LOW":    "Thấp",
	"MED":    "Trung bình",
	"HIGH":   "Cao",
	"URGENT": "Khẩn cấp",
}

func PriorityDisplay(value string) string {
	if v, ok := priorityLabels[value]; ok {
		return v
	}
	return value
}

func FieldLabel(field string) string {
	switch field {
	case "priority":
		return "Độ ưu tiên"
	case "due_date":
		return "Hạn chót"
	case "start_date":
		return "Ngày bắt đầu"
	case "title":
		return "Tiêu đề"
	case "description":
		return "Mô tả"
	case "name":
		return "Tên"
	case "domain":
		return "Domain"
	case "icon":
		return "Icon"
	case "background":
		return "Nền"
	default:
		return field
	}
}

func FormatDisplayValue(field string, value interface{}) string {
	if value == nil {
		return "(trống)"
	}
	switch field {
	case "priority":
		s, ok := value.(string)
		if !ok {
			return "(trống)"
		}
		return PriorityDisplay(s)
	case "due_date", "start_date":
		s, ok := value.(string)
		if !ok {
			return "(trống)"
		}
		t, err := time.Parse(time.RFC3339, s)
		if err != nil {
			return s
		}
		return t.Format("02/01/2006")
	case "description":
		s, ok := value.(string)
		if !ok || s == "" {
			return "(trống)"
		}
		return "(có nội dung)"
	case "title", "name", "icon", "background", "domain":
		s, ok := value.(string)
		if !ok || s == "" {
			return "(trống)"
		}
		return s
	default:
		s, ok := value.(string)
		if !ok {
			return "(trống)"
		}
		return s
	}
}

func BuildChangeField(field string, oldValue, newValue interface{}) ChangeField {
	return ChangeField{
		Field:    field,
		OldValue: oldValue,
		NewValue: newValue,
		Display: &DisplayInfo{
			FieldLabel: FieldLabel(field),
			OldDisplay: FormatDisplayValue(field, oldValue),
			NewDisplay: FormatDisplayValue(field, newValue),
		},
	}
}
