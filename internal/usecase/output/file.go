package output

import "time"

type FileDetail struct {
	ID         int64
	ObjectKey  string
	FileName   string
	FileSize   int64
	MimeType   string
	Usage      string
	ResourceID *int64
	UploaderID int64
	CreatedAt  time.Time
}
