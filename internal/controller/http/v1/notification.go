package v1

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v3"
	sharedresp "github.com/scc749/nimbus-blog-api/internal/controller/http/shared"
	"github.com/scc749/nimbus-blog-api/internal/controller/http/v1/response"
	authUC "github.com/scc749/nimbus-blog-api/internal/usecase/auth/user"
	"github.com/scc749/nimbus-blog-api/internal/usecase/input"
)

// @Summary 通知列表
// @Tags V1.Notifications
// @Produce json
// @Security BearerAuth
// @Param page query int false "页码" default(1)
// @Param page_size query int false "分页大小" default(10)
// @Param sort_by query string false "排序字段" Enums(created_at)
// @Param order query string false "排序方向" Enums(asc,desc) default(desc)
// @Param filter.is_read query string false "是否已读过滤"
// @Success 200 {object} sharedresp.Envelope{data=response.NotificationDetailPage}
// @Failure 401 {object} sharedresp.Envelope
// @Failure 500 {object} sharedresp.Envelope
// @Router /v1/notifications [get]
func (r *V1) listNotifications(ctx fiber.Ctx) error {
	claims, ok := authUC.AccessClaimsFromContext(ctx.Context())
	if !ok || claims == nil {
		return sharedresp.WriteError(ctx, http.StatusUnauthorized, response.ErrorLoginRequired, "login required")
	}
	uid, err := claims.UserIDInt()
	if err != nil {
		return sharedresp.WriteError(ctx, http.StatusUnauthorized, response.ErrorTokenInvalid, "invalid token")
	}

	pq := sharedresp.ParsePageQueryWithOptions(ctx, sharedresp.WithAllowedSortBy("created_at"), sharedresp.WithAllowedFilters("is_read"))
	pageParams := input.PageParams{
		Page:     pq.Page,
		PageSize: pq.PageSize,
	}
	var sortParams *input.SortParams
	if pq.SortBy != "" {
		sortParams = &input.SortParams{
			SortBy: pq.SortBy,
			Order:  pq.Order,
		}
	}
	var isRead input.BoolFilterParam
	if s, ok := pq.Filters["is_read"]; ok && s != "" {
		isRead = input.ParseBoolFilterParam(s)
	}

	result, err := r.notification.ListMyNotifications(ctx.Context(), input.ListNotifications{
		PageParams: pageParams,
		Sort:       sortParams,
		UserID:     uid,
		IsRead:     isRead,
	})
	if err != nil {
		r.logger.Error(err, "http - v1 - notification - listNotifications - usecase")
		return sharedresp.WriteError(ctx, http.StatusInternalServerError, response.ErrorListNotificationsFailed, "failed to list notifications")
	}

	list := make([]response.NotificationDetail, 0, len(result.Items))
	for _, n := range result.Items {
		list = append(list, response.NotificationDetail{
			ID:        n.ID,
			Type:      n.Type,
			Title:     n.Title,
			Content:   n.Content,
			Meta:      n.Meta,
			PostSlug:  n.PostSlug,
			CommentID: n.CommentID,
			TargetURL: n.TargetURL,
			IsRead:    n.IsRead,
			CreatedAt: n.CreatedAt,
		})
	}
	return sharedresp.WriteSuccess(ctx, sharedresp.WithData(sharedresp.NewPage(list, result.Page, result.PageSize, result.Total)))
}

// @Summary 未读通知数量
// @Tags V1.Notifications
// @Produce json
// @Security BearerAuth
// @Success 200 {object} sharedresp.Envelope{data=response.UnreadCount}
// @Failure 401 {object} sharedresp.Envelope
// @Failure 500 {object} sharedresp.Envelope
// @Router /v1/notifications/unread [get]
func (r *V1) getUnreadCount(ctx fiber.Ctx) error {
	claims, ok := authUC.AccessClaimsFromContext(ctx.Context())
	if !ok || claims == nil {
		return sharedresp.WriteError(ctx, http.StatusUnauthorized, response.ErrorLoginRequired, "login required")
	}
	uid, err := claims.UserIDInt()
	if err != nil {
		return sharedresp.WriteError(ctx, http.StatusUnauthorized, response.ErrorTokenInvalid, "invalid token")
	}

	count, err := r.notification.GetUnreadCount(ctx.Context(), uid)
	if err != nil {
		r.logger.Error(err, "http - v1 - notification - getUnreadCount - usecase")
		return sharedresp.WriteError(ctx, http.StatusInternalServerError, response.ErrorGetUnreadCountFailed, "failed to get unread count")
	}
	return sharedresp.WriteSuccess(ctx, sharedresp.WithData(response.UnreadCount{Count: count}))
}

// @Summary 标记通知已读
// @Tags V1.Notifications
// @Produce json
// @Security BearerAuth
// @Param id path int true "通知 ID"
// @Success 200 {object} sharedresp.Envelope
// @Failure 400 {object} sharedresp.Envelope
// @Failure 401 {object} sharedresp.Envelope
// @Failure 500 {object} sharedresp.Envelope
// @Router /v1/notifications/{id}/read [put]
func (r *V1) markRead(ctx fiber.Ctx) error {
	claims, ok := authUC.AccessClaimsFromContext(ctx.Context())
	if !ok || claims == nil {
		return sharedresp.WriteError(ctx, http.StatusUnauthorized, response.ErrorLoginRequired, "login required")
	}
	uid, err := claims.UserIDInt()
	if err != nil {
		return sharedresp.WriteError(ctx, http.StatusUnauthorized, response.ErrorTokenInvalid, "invalid token")
	}

	id := ctx.Params("id")
	if id == "" {
		return sharedresp.WriteError(ctx, http.StatusBadRequest, response.ErrorParamMissing, "missing id")
	}
	nid, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return sharedresp.WriteError(ctx, http.StatusBadRequest, response.ErrorParamFormat, "invalid id")
	}

	if err := r.notification.MarkRead(ctx.Context(), nid, uid); err != nil {
		r.logger.Error(err, "http - v1 - notification - markRead - usecase")
		return sharedresp.WriteError(ctx, http.StatusInternalServerError, response.ErrorMarkReadFailed, "mark read failed")
	}
	return sharedresp.WriteSuccess(ctx)
}

// @Summary 标记全部通知已读
// @Tags V1.Notifications
// @Produce json
// @Security BearerAuth
// @Success 200 {object} sharedresp.Envelope
// @Failure 401 {object} sharedresp.Envelope
// @Failure 500 {object} sharedresp.Envelope
// @Router /v1/notifications/read-all [put]
func (r *V1) markAllRead(ctx fiber.Ctx) error {
	claims, ok := authUC.AccessClaimsFromContext(ctx.Context())
	if !ok || claims == nil {
		return sharedresp.WriteError(ctx, http.StatusUnauthorized, response.ErrorLoginRequired, "login required")
	}
	uid, err := claims.UserIDInt()
	if err != nil {
		return sharedresp.WriteError(ctx, http.StatusUnauthorized, response.ErrorTokenInvalid, "invalid token")
	}

	if err := r.notification.MarkAllRead(ctx.Context(), uid); err != nil {
		r.logger.Error(err, "http - v1 - notification - markAllRead - usecase")
		return sharedresp.WriteError(ctx, http.StatusInternalServerError, response.ErrorMarkAllReadFailed, "mark all read failed")
	}
	return sharedresp.WriteSuccess(ctx)
}

// @Summary 删除通知
// @Tags V1.Notifications
// @Produce json
// @Security BearerAuth
// @Param id path int true "通知 ID"
// @Success 200 {object} sharedresp.Envelope
// @Failure 400 {object} sharedresp.Envelope
// @Failure 401 {object} sharedresp.Envelope
// @Failure 500 {object} sharedresp.Envelope
// @Router /v1/notifications/{id} [delete]
func (r *V1) deleteNotification(ctx fiber.Ctx) error {
	claims, ok := authUC.AccessClaimsFromContext(ctx.Context())
	if !ok || claims == nil {
		return sharedresp.WriteError(ctx, http.StatusUnauthorized, response.ErrorLoginRequired, "login required")
	}
	uid, err := claims.UserIDInt()
	if err != nil {
		return sharedresp.WriteError(ctx, http.StatusUnauthorized, response.ErrorTokenInvalid, "invalid token")
	}

	id := ctx.Params("id")
	if id == "" {
		return sharedresp.WriteError(ctx, http.StatusBadRequest, response.ErrorParamMissing, "missing id")
	}
	nid, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return sharedresp.WriteError(ctx, http.StatusBadRequest, response.ErrorParamFormat, "invalid id")
	}

	if err := r.notification.DeleteNotification(ctx.Context(), nid, uid); err != nil {
		r.logger.Error(err, "http - v1 - notification - deleteNotification - usecase")
		return sharedresp.WriteError(ctx, http.StatusInternalServerError, response.ErrorDeleteNotificationFailed, "delete notification failed")
	}
	return sharedresp.WriteSuccess(ctx)
}

// @Summary 通知 SSE 流
// @Description 使用 Query Token 鉴权（token=access_token），返回 text/event-stream。
// @Tags V1.Notifications
// @Param token query string true "Access Token"
// @Success 200 {string} string "SSE stream"
// @Failure 401 {object} sharedresp.Envelope
// @Router /v1/notifications/stream [get]
func (r *V1) streamNotifications(ctx fiber.Ctx) error {
	tokenStr := ctx.Query("token", "")
	if tokenStr == "" {
		return sharedresp.WriteError(ctx, http.StatusUnauthorized, response.ErrorLoginRequired, "missing token")
	}
	claims, err := r.signer.ParseAccess(tokenStr)
	if err != nil {
		return sharedresp.WriteError(ctx, http.StatusUnauthorized, response.ErrorTokenInvalid, "invalid token")
	}
	uid, err := claims.UserIDInt()
	if err != nil {
		return sharedresp.WriteError(ctx, http.StatusUnauthorized, response.ErrorTokenInvalid, "invalid token")
	}

	ctx.Set("Content-Type", "text/event-stream")
	ctx.Set("Cache-Control", "no-cache")
	ctx.Set("Connection", "keep-alive")
	ctx.Set("X-Accel-Buffering", "no")

	ch := r.notification.Subscribe(uid)

	ctx.RequestCtx().SetBodyStreamWriter(func(w *bufio.Writer) {
		defer r.notification.Unsubscribe(uid, ch)

		count, _ := r.notification.GetUnreadCount(context.Background(), uid)
		countData, _ := json.Marshal(map[string]int64{"count": count})
		fmt.Fprintf(w, "event: unread_count\ndata: %s\n\n", countData)
		w.Flush()

		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case event, ok := <-ch:
				if !ok {
					return
				}
				if event.Name != "" {
					fmt.Fprintf(w, "event: %s\n", event.Name)
				}
				fmt.Fprintf(w, "data: %s\n\n", event.Data)
				if err := w.Flush(); err != nil {
					return
				}
			case <-ticker.C:
				fmt.Fprintf(w, ": heartbeat\n\n")
				if err := w.Flush(); err != nil {
					return
				}
			}
		}
	})
	return nil
}
