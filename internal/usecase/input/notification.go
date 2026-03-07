package input

type ListNotifications struct {
	PageParams
	Sort   *SortParams
	UserID int64
	IsRead BoolFilterParam
}

type SendAdminNotification struct {
	UserID  int64
	Title   string
	Content string
}
