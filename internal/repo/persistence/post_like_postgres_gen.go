package persistence

import (
	"context"
	"errors"

	"github.com/scc749/nimbus-blog-api/internal/repo"
	"github.com/scc749/nimbus-blog-api/internal/repo/persistence/gen/model"
	"github.com/scc749/nimbus-blog-api/internal/repo/persistence/gen/query"
	"gorm.io/gorm"
)

type postLikeRepo struct {
	query *query.Query
}

func NewPostLikeRepo(db *gorm.DB) repo.PostLikeRepo {
	return &postLikeRepo{query: query.Use(db)}
}

func (r *postLikeRepo) Toggle(ctx context.Context, postID, userID int64) (liked bool, count int32, err error) {
	err = r.query.Transaction(func(tx *query.Query) error {
		pl := tx.PostLike
		p := tx.Post

		_, findErr := pl.WithContext(ctx).Where(pl.PostID.Eq(postID), pl.UserID.Eq(userID)).First()
		if findErr != nil && !errors.Is(findErr, gorm.ErrRecordNotFound) {
			return findErr
		}

		if errors.Is(findErr, gorm.ErrRecordNotFound) {
			if err := pl.WithContext(ctx).Create(&model.PostLike{PostID: postID, UserID: userID}); err != nil {
				return err
			}
			if _, err := p.WithContext(ctx).Where(p.ID.Eq(postID)).UpdateSimple(p.Likes.Add(1)); err != nil {
				return err
			}
			liked = true
		} else {
			if _, err := pl.WithContext(ctx).Where(pl.PostID.Eq(postID), pl.UserID.Eq(userID)).Delete(); err != nil {
				return err
			}
			if _, err := p.WithContext(ctx).Where(p.ID.Eq(postID)).UpdateSimple(p.Likes.Sub(1)); err != nil {
				return err
			}
			liked = false
		}

		mp, err := p.WithContext(ctx).Select(p.Likes).Where(p.ID.Eq(postID)).First()
		if err != nil {
			return err
		}
		count = mp.Likes
		return nil
	})
	return
}

func (r *postLikeRepo) HasLiked(ctx context.Context, postID, userID int64) (bool, error) {
	pl := r.query.PostLike
	_, err := pl.WithContext(ctx).Where(pl.PostID.Eq(postID), pl.UserID.Eq(userID)).First()
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (r *postLikeRepo) Remove(ctx context.Context, postID, userID int64) (removed bool, count int32, err error) {
	err = r.query.Transaction(func(tx *query.Query) error {
		pl := tx.PostLike
		p := tx.Post

		info, delErr := pl.WithContext(ctx).Where(pl.PostID.Eq(postID), pl.UserID.Eq(userID)).Delete()
		if delErr != nil {
			return delErr
		}

		if info.RowsAffected > 0 {
			if _, err := p.WithContext(ctx).Where(p.ID.Eq(postID)).UpdateSimple(p.Likes.Sub(1)); err != nil {
				return err
			}
			removed = true
		}

		mp, err := p.WithContext(ctx).Select(p.Likes).Where(p.ID.Eq(postID)).First()
		if err != nil {
			return err
		}
		count = mp.Likes
		return nil
	})
	return
}
