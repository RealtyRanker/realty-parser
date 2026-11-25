package worker

import (
	"context"
	"errors"
	"maps"
	"slices"

	"go.uber.org/zap"

	"github.com/asmisnik/realty-parser/internal/domclick"
	"github.com/asmisnik/realty-parser/internal/metrics"
)

func (w *Worker) runDomclick(ctx context.Context) int {
	regions := w.cfg.Domclick.Regions
	filters := w.cfg.Search
	total := 0
	for _, region := range slices.Sorted(maps.Keys(regions)) {
		for _, dealType := range []string{"rent", "sale"} {
			if ctx.Err() != nil {
				return total
			}
			log := w.logger.With(zap.String("source", "domclick"), zap.Int("region", region), zap.String("deal_type", dealType))
			inserted := 0
			for page := 1; page <= filters.PagesLimit && inserted < filters.OverallLimit && ctx.Err() == nil; page++ {
				result, err := w.domclick.Search(ctx, filters, region, regions[region], dealType, page)
				if err != nil {
					metrics.FetchErrors.Inc()
					log.Warn("Domclick search failed", zap.Int("page", page), zap.Error(err))
					if errors.Is(err, domclick.ErrBlocked) {
						return total + inserted
					}
					break
				}
				metrics.PagesFetched.Inc()
				metrics.FlatsFound.Add(float64(len(result.Flats)))
				log.Info("found flats on search page", zap.Int("page", page), zap.Int("count", len(result.Flats)))
				for _, flat := range result.Flats {
					if inserted >= filters.OverallLimit || ctx.Err() != nil {
						break
					}
					stored, err := w.storeAndPublish(ctx, flat)
					if err != nil {
						log.Warn("skipping flat", zap.String("href", flat.Link), zap.Error(err))
					}
					if stored {
						inserted++
					}
				}
				if !result.HasMore {
					break
				}
			}
			total += inserted
		}
	}
	return total
}
