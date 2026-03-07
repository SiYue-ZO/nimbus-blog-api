package output

import "time"

type FileDetail struct {
	ID         int64     `json:"id"`
	ObjectKey  string    `json:"object_key"`
	FileName   string    `json:"file_name"`
	FileSize   int64     `json:"file_size"`
	MimeType   string    `json:"mime_type"`
	Usage      string    `json:"usage"`
	ResourceID *int64    `json:"resource_id"`
	UploaderID int64     `json:"uploader_id"`
	CreatedAt  time.Time `json:"created_at"`
}
