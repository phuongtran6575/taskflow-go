package helper

import "strings"

func SplitComma(s string) []string {
	values := strings.Split(s, ",")
	for i := range values {
		values[i] = strings.TrimSpace(values[i])
	}
	return values
}

func CheckAlreadyExistPermissions(existPermissionIds []string, permissionIDs []string) (exists []string, news []string) {
	set := make(map[string]struct{}, len(existPermissionIds))

	for _, id := range existPermissionIds {
		set[id] = struct{}{}
	}

	for _, id := range permissionIDs {
		if _, ok := set[id]; ok {
			exists = append(exists, id)
		} else {
			news = append(news, id)
		}
	}

	return exists, news
}
