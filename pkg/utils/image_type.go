package utils

var ValidImageTypes = []string{"micro", "minimal", "platform", "init"}

func IsValidImageType(t string) bool {
	for _, v := range ValidImageTypes {
		if t == v {
			return true
		}
	}
	return false
}
