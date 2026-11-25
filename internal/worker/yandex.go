package worker

import (
	"context"
	"errors"

	"go.uber.org/zap"

	"github.com/asmisnik/realty-parser/internal/db"
	"github.com/asmisnik/realty-parser/internal/metrics"
	"github.com/asmisnik/realty-parser/internal/yandex"
)

func (w *Worker) runYandex(ctx context.Context, combos []db.RegionDealType) int {
	total := 0
	filters := w.cfg.Search
	if filters.OverallLimit <= 0 || filters.PagesLimit <= 0 {
		return 0
	}
	for _, combo := range combos {
		rgid := w.cfg.Yandex.Regions[combo.Region]
		if rgid <= 0 {
			continue
		}
		if ctx.Err() != nil {
			break
		}
		log := w.logger.With(zap.String("source", "yandex"), zap.Int("region", combo.Region), zap.String("deal_type", combo.DealType))
		flats, err := w.yandex.Search(ctx, filters, combo.Region, rgid, combo.DealType)
		if err != nil {
			metrics.FetchErrors.Inc()
			log.Warn("Yandex Realty search failed", zap.Error(err))
			if errors.Is(err, yandex.ErrBlocked) {
				break
			}
			continue
		}
		metrics.PagesFetched.Inc()
		metrics.FlatsFound.Add(float64(len(flats)))
		log.Info("found flats on search page", zap.Int("count", len(flats)))
		inserted := 0
		for _, f := range flats {
			if inserted >= filters.OverallLimit || ctx.Err() != nil {
				break
			}
			stored, err := w.storeAndPublish(ctx, f)
			if err != nil {
				log.Warn("skipping flat", zap.String("href", f.Link), zap.Error(err))
			}
			if stored {
				inserted++
			}
		}
		total += inserted
	}
	return total
}
