package utils

import (
	"path"
	"strings"
)

var contentTypes = map[string]string{
	".pdf":  "application/pdf",
	".docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".png":  "image/png",
	".gif":  "image/gif",
}

func GetContentType(fileName string) (string, bool) {
	ext := strings.ToLower(path.Ext(fileName))
	contentType, exists := contentTypes[ext]
	return contentType, exists
}
