package response

import sharedresp "github.com/scc749/nimbus-blog-api/internal/controller/http/shared"

type PostSummaryPage = sharedresp.Page[PostSummary]
type NotificationDetailPage = sharedresp.Page[NotificationDetail]
