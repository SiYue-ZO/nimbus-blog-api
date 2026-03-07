package viewbuffer

import (
	"context"
	"time"

	"github.com/scc749/nimbus-blog-api/internal/entity"
	"github.com/scc749/nimbus-blog-api/internal/repo"
	"github.com/scc749/nimbus-blog-api/internal/repo/persistence/gen/model"
	"github.com/scc749/nimbus-blog-api/internal/repo/persistence/gen/query"
	"github.com/scc749/nimbus-blog-api/pkg/logger"
	"gorm.io/gorm"
)

const (
	_chanSize      = 4096
	_flushInterval = 30 * time.Second
	_dedupeWindow  = 30 * time.Minute
	_maxBatchSize  = 500
	_batchInsert   = 100
)

var _ repo.PostViewRepo = (*bufferedPostViewRepo)(nil)

type bufferedPostViewRepo struct {
	query  *query.Query
	logger logger.Interface
	ch     chan viewEvent
	cancel context.CancelFunc
	done   chan struct{}
}

type viewEvent struct {
	postID    int64
	ip        string
	userAgent *string
	referer   *string
	timestamp time.Time
}

type dedupeKey struct {
	postID int64
	ip     string
}

func New(db *gorm.DB, l logger.Interface) (repo.PostViewRepo, func()) {
	ctx, cancel := context.WithCancel(context.Background())
	r := &bufferedPostViewRepo{
		query:  query.Use(db),
		logger: l,
		ch:     make(chan viewEvent, _chanSize),
		cancel: cancel,
		done:   make(chan struct{}),
	}
	go r.flushLoop(ctx)
	cleanup := func() {
		r.cancel()
		<-r.done
	}
	return r, cleanup
}

func (r *bufferedPostViewRepo) Record(_ context.Context, pv entity.PostView) error {
	select {
	case r.ch <- viewEvent{
		postID:    pv.PostID,
		ip:        pv.IPAddress,
		userAgent: pv.UserAgent,
		referer:   pv.Referer,
		timestamp: pv.ViewedAt,
	}:
	default:
	}
	return nil
}

func (r *bufferedPostViewRepo) flushLoop(ctx context.Context) {
	defer close(r.done)

	ticker := time.NewTicker(_flushInterval)
	defer ticker.Stop()

	seen := make(map[dedupeKey]time.Time)
	var pending []viewEvent
	deltas := make(map[int64]int32)

	flush := func() {
		if len(pending) == 0 {
			return
		}
		r.writeBatch(pending, deltas)
		pending = pending[:0]
		for k := range deltas {
			delete(deltas, k)
		}
	}

	for {
		select {
		case ev := <-r.ch:
			key := dedupeKey{postID: ev.postID, ip: ev.ip}
			if last, ok := seen[key]; ok && time.Since(last) < _dedupeWindow {
				continue
			}
			seen[key] = ev.timestamp
			pending = append(pending, ev)
			deltas[ev.postID]++
			if len(pending) >= _maxBatchSize {
				flush()
			}

		case <-ticker.C:
			flush()
			now := time.Now()
			for k, t := range seen {
				if now.Sub(t) > _dedupeWindow {
					delete(seen, k)
				}
			}

		case <-ctx.Done():
		drain:
			for {
				select {
				case ev := <-r.ch:
					key := dedupeKey{postID: ev.postID, ip: ev.ip}
					if last, ok := seen[key]; ok && time.Since(last) < _dedupeWindow {
						continue
					}
					seen[key] = ev.timestamp
					pending = append(pending, ev)
					deltas[ev.postID]++
				default:
					break drain
				}
			}
			flush()
			return
		}
	}
}

func (r *bufferedPostViewRepo) writeBatch(events []viewEvent, deltas map[int64]int32) {
	ctx := context.Background()

	models := make([]*model.PostView, len(events))
	for i, ev := range events {
		models[i] = &model.PostView{
			PostID:    ev.postID,
			IPAddress: ev.ip,
			UserAgent: ev.userAgent,
			Referer:   ev.referer,
			ViewedAt:  ev.timestamp,
		}
	}
	if err := r.query.PostView.WithContext(ctx).CreateInBatches(models, _batchInsert); err != nil {
		r.logger.Error(err, "viewbuffer - writeBatch - CreateInBatches")
	}

	p := r.query.Post
	for postID, delta := range deltas {
		if _, err := p.WithContext(ctx).Where(p.ID.Eq(postID)).UpdateSimple(p.Views.Add(delta)); err != nil {
			r.logger.Error(err, "viewbuffer - writeBatch - IncrementViews")
		}
	}
}
