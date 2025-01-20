package enum

import (
	types "boilerplate-go/internal/common/type"
)

type FileTypeEnum string

const (
	IMAGE FileTypeEnum = "image"
	VIDEO FileTypeEnum = "video"
	FILE  FileTypeEnum = "file"
)

func (e FileTypeEnum) ToString() string {
	switch e {
	case IMAGE:
		return "image"
	case FILE:
		return "file"
	case VIDEO:
		return "video"
	default:
		return ""
	}
}

func (e FileTypeEnum) IsValid() bool {
	switch e {
	case IMAGE, FILE, VIDEO:
		return true
	}

	return false
}

func (e FileTypeEnum) IsValidImage(file *types.BufferedFile) bool {
	if e == IMAGE {
		switch file.MimeType {
		case "image/jpeg", "image/png", "image/gif", "image/bmp", "image/webp", "image/tiff", "image/svg+xml", "image/x-icon", "image/heic", "image/heif":
			return true
		}
	}

	return false
}

func (e FileTypeEnum) IsValidVideo(file *types.BufferedFile) bool {
	if e == VIDEO {
		switch file.MimeType {
		case "video/mp4", "video/webm", "video/ogg", "video/avi", "video/mkv", "video/quicktime", "video/x-flv", "video/x-msvideo":
			return true
		}
	}

	return false
}
