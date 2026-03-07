package response

import sharedresp "github.com/scc749/nimbus-blog-api/internal/controller/http/shared"

type FileDetailPage = sharedresp.Page[FileDetail]
type UserDetailPage = sharedresp.Page[UserDetail]
type PostSummaryPage = sharedresp.Page[PostSummary]
type CategoryDetailPage = sharedresp.Page[CategoryDetail]
type TagDetailPage = sharedresp.Page[TagDetail]
type CommentDetailPage = sharedresp.Page[CommentDetail]
type FeedbackDetailPage = sharedresp.Page[FeedbackDetail]
type LinkDetailPage = sharedresp.Page[LinkDetail]
type SiteSettingDetailPage = sharedresp.Page[SiteSettingDetail]
