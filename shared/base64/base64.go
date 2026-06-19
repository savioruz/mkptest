package base64

import "strings"

func GetContentType(file string) string {
	start := len("data:")
	end := strings.Index(file, ";base64,")

	if end == -1 || end < start {
		return ""
	}

	return file[start:end]
}
