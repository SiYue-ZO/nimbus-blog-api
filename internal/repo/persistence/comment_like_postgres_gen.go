package persistence

import (
	"context"
	"errors"

	"github.com/scc749/nimbus-blog-api/internal/repo"
	"github.com/scc749/nimbus-blog-api/internal/repo/persistence/gen/model"
	"github.com/scc749/nimbus-blog-api/internal/repo/persistence/gen/query"
	"gorm.io/gorm"
)

type commentLikeRepo struct {
	query *query.Query
}

func NewCommentLikeRepo(db *gorm.DB) repo.CommentLikeRepo {
	return &commentLikeRepo{query: query.Use(db)}
}

func (r *commentLikeRepo) Toggle(ctx context.Context, commentID, userID int64) (liked bool, count int32, err error) {
	err = r.query.Transaction(func(tx *query.Query) error {
		cl := tx.CommentLike
		c := tx.Comment

		_, findErr := cl.WithContext(ctx).Where(cl.CommentID.Eq(commentID), cl.UserID.Eq(userID)).First()
		if findErr != nil && !errors.Is(findErr, gorm.ErrRecordNotFound) {
			return findErr
		}

		if errors.Is(findErr, gorm.ErrRecordNotFound) {
			if err := cl.WithContext(ctx).Create(&model.CommentLike{CommentID: commentID, UserID: userID}); err != nil {
				return err
			}
			if _, err := c.WithContext(ctx).Where(c.ID.Eq(commentID)).UpdateSimple(c.Likes.Add(1)); err != nil {
				return err
			}
			liked = true
		} else {
			if _, err := cl.WithContext(ctx).Where(cl.CommentID.Eq(commentID), cl.UserID.Eq(userID)).Delete(); err != nil {
				return err
			}
			if _, err := c.WithContext(ctx).Where(c.ID.Eq(commentID)).UpdateSimple(c.Likes.Sub(1)); err != nil {
				return err
			}
			liked = false
		}

		mc, err := c.WithContext(ctx).Select(c.Likes).Where(c.ID.Eq(commentID)).First()
		if err != nil {
			return err
		}
		count = mc.Likes
		return nil
	})
	return
}

func (r *commentLikeRepo) Remove(ctx context.Context, commentID, userID int64) (removed bool, count int32, err error) {
	err = r.query.Transaction(func(tx *query.Query) error {
		cl := tx.CommentLike
		c := tx.Comment

		info, delErr := cl.WithContext(ctx).Where(cl.CommentID.Eq(commentID), cl.UserID.Eq(userID)).Delete()
		if delErr != nil {
			return delErr
		}

		if info.RowsAffected > 0 {
			if _, err := c.WithContext(ctx).Where(c.ID.Eq(commentID)).UpdateSimple(c.Likes.Sub(1)); err != nil {
				return err
			}
			removed = true
		}

		mc, err := c.WithContext(ctx).Select(c.Likes).Where(c.ID.Eq(commentID)).First()
		if err != nil {
			return err
		}
		count = mc.Likes
		return nil
	})
	return
}

func (r *commentLikeRepo) HasLiked(ctx context.Context, commentID, userID int64) (bool, error) {
	cl := r.query.CommentLike
	_, err := cl.WithContext(ctx).Where(cl.CommentID.Eq(commentID), cl.UserID.Eq(userID)).First()
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
