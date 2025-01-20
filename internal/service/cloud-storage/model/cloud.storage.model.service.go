package model

import (
	"boilerplate-go/internal/common/enum"
)

type ResultDownload struct {
	URL            string `json:"url" validate:"required"`
	OriginFileName string `json:"originFileName" validate:"required"`
	FileName       string `json:"fileName" validate:"required"`
	MimeType       string `json:"mimeType" validate:"required"`
	Size           int    `json:"size" validate:"required"`
	Token          string `json:"token" validate:"required"`
}

type UploadPost struct {
	Folder    string            `json:"folder" validate:"required"`
	Directory string            `json:"directory" validate:"required"`
	MediaType enum.FileTypeEnum `json:"media" validate:"required,enum"`
	IsMessage bool              `json:"isMessage" validate:"required"`
	Caption   string            `json:"caption" validate:"required"`
}
