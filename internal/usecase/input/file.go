package input

type SaveFileMeta struct {
	ObjectKey  string
	FileName   string
	FileSize   int64
	MimeType   string
	Usage      string
	ResourceID *int64
	UploaderID int64
}

type ListFiles struct {
	PageParams
	Sort  *SortParams
	Usage StringFilterParam
}
