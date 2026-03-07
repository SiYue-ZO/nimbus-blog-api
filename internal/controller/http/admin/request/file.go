package request

type GenerateUploadURL struct {
	UploadType    string `json:"upload_type" validate:"required,oneof=avatar post_cover post_content"`
	ContentType   string `json:"content_type" validate:"required,oneof=image/jpeg image/png image/webp"`
	ResourceID    int64  `json:"resource_id" validate:"omitempty,gte=0"`
	FileName      string `json:"file_name" validate:"required,min=1,max=255"`
	FileSize      int64  `json:"file_size" validate:"required,gte=1,lte=10485760"`
	ExpirySeconds int    `json:"expiry_seconds" validate:"omitempty,gte=1,lte=86400"`
}
