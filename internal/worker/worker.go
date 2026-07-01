package worker

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"

	fhttp "github.com/bogdanfinn/fhttp"
	tls_client "github.com/bogdanfinn/tls-client"
	"github.com/bogdanfinn/tls-client/profiles"
	"go.uber.org/zap"

	"github.com/asmisnik/realty-parser/internal/cian"
	"github.com/asmisnik/realty-parser/internal/config"
	"github.com/asmisnik/realty-parser/internal/db"
	"github.com/asmisnik/realty-parser/internal/kafka"
	"github.com/asmisnik/realty-parser/internal/metrics"
	"github.com/asmisnik/realty-parser/internal/model"
)

type Worker struct {
	cfg      *config.Config
	db       *db.DB
	logger   *zap.Logger
	client   tls_client.HttpClient
	producer *kafka.Producer
}

func New(cfg *config.Config, database *db.DB, logger *zap.Logger) *Worker {
	jar := tls_client.NewCookieJar()

	if cfg.Worker.CookieHeader != "" {
		cianURL, _ := url.Parse("https://www.cian.ru")
		dummy := &fhttp.Request{Header: fhttp.Header{"Cookie": []string{cfg.Worker.CookieHeader}}}
		jar.SetCookies(cianURL, dummy.Cookies())
	}

	client, _ := tls_client.NewHttpClient(tls_client.NewNoopLogger(),
		tls_client.WithTimeoutSeconds(cfg.Worker.RequestTimeoutMs/1000),
		tls_client.WithClientProfile(profiles.Chrome_124),
		tls_client.WithCookieJar(jar),
	)

	return &Worker{
		cfg:      cfg,
		db:       database,
		logger:   logger,
		client:   client,
		producer: kafka.NewProducer(cfg.Kafka.Brokers, cfg.Kafka.Topic),
	}
}

// Start runs the worker loop until ctx is cancelled. It executes immediately on start,
// then repeats every cfg.Worker.PeriodSeconds.
func (w *Worker) Start(ctx context.Context) {
	w.runOnce(ctx)

	ticker := time.NewTicker(time.Duration(w.cfg.Worker.PeriodSeconds) * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.runOnce(ctx)
		}
	}
}

func (w *Worker) runOnce(ctx context.Context) {
	start := time.Now()
	metrics.LastRunTimestamp.SetToCurrentTime()
	w.logger.Info("worker run started")

	processed := w.doRun(ctx)

	elapsed := time.Since(start).Seconds()
	metrics.LastRunDuration.Set(elapsed)
	metrics.LastRunFlatsProcessed.Set(float64(processed))
	w.logger.Info("worker run finished",
		zap.Float64("duration_s", elapsed),
		zap.Int("flats_processed", processed),
	)
}

// defaultRegion is used when no active subscriptions exist yet (e.g. on a fresh deployment).
const defaultRegion = 1 // Москва

func (w *Worker) doRun(ctx context.Context) int {
	regions, err := w.db.GetDistinctRegions(ctx)
	if err != nil {
		w.logger.Error("fetching distinct regions failed", zap.Error(err))
		return 0
	}
	if len(regions) == 0 {
		regions = []int{defaultRegion}
	}

	parsed := 0
	for _, region := range regions {
		parsed += w.runRegion(ctx, region)
	}
	return parsed
}

func (w *Worker) runRegion(ctx context.Context, region int) int {
	cfg := w.cfg.Search
	parsed := 0

	for page := 1; parsed < cfg.OverallLimit && page <= cfg.PagesLimit; page++ {
		searchURL := cian.BuildSearchURL(cfg, region, page)
		w.logger.Debug("fetching search page", zap.Int("region", region), zap.String("url", searchURL))

		body, err := w.fetchURL(ctx, searchURL, "https://www.cian.ru/")
		if err != nil {
			w.logger.Error("search page fetch failed", zap.Int("region", region), zap.Int("page", page), zap.Error(err))
			metrics.FetchErrors.Inc()
			continue
		}
		metrics.PagesFetched.Inc()

		hrefs, err := cian.ParseSearchPageHrefs(body, w.logger)
		if err != nil {
			w.logger.Error("search page parse failed", zap.Int("region", region), zap.Int("page", page), zap.Error(err))
			metrics.ParseErrors.Inc()
			continue
		}
		w.logger.Info("found flats on search page", zap.Int("region", region), zap.Int("page", page), zap.Int("count", len(hrefs)))
		metrics.FlatsFound.Add(float64(len(hrefs)))

		for _, href := range hrefs {
			if parsed >= cfg.OverallLimit {
				break
			}
			if err := w.processFlat(ctx, href, region); err != nil {
				w.logger.Warn("skipping flat", zap.String("href", href), zap.Error(err))
				continue
			}
			parsed++
		}
	}
	return parsed
}

func (w *Worker) processFlat(ctx context.Context, href string, region int) error {
	exists, err := w.db.FlatExists(ctx, href)
	if err != nil {
		return fmt.Errorf("db exists check: %w", err)
	}
	if exists {
		w.logger.Debug("flat already in db", zap.String("href", href))
		return nil
	}

	info, err := w.fetchAndParseFlatWithRetry(ctx, href, region)
	if err != nil {
		metrics.ParseErrors.Inc()
		return err
	}

	if err := w.db.InsertFlat(ctx, info); err != nil {
		return fmt.Errorf("db insert: %w", err)
	}

	metrics.FlatsInserted.Inc()
	metrics.FlatScores.Observe(float64(info.FlatScore))
	metrics.FlatPrices.Observe(float64(info.Price))
	w.logger.Info("flat inserted",
		zap.String("href", href),
		zap.Int("score", info.FlatScore),
		zap.Int("price", info.Price),
		zap.Int("rooms", info.RoomNumber),
		zap.Float64("area", info.TotalArea),
	)

	if err := w.producer.PublishFlat(ctx, info); err != nil {
		w.logger.Warn("kafka publish failed", zap.String("href", href), zap.Error(err))
	} else {
		w.logger.Debug("flat published to kafka", zap.String("href", href))
	}

	return nil
}

func (w *Worker) fetchAndParseFlatWithRetry(ctx context.Context, href string, region int) (*model.FlatInfo, error) {
	time.Sleep(time.Duration(w.cfg.Worker.SleepBeforeRequestMs) * time.Millisecond)

	sleepMs := float64(w.cfg.Worker.RetrySleepMs)
	var lastErr error

	for try := 0; try <= w.cfg.Worker.MaxRetries; try++ {
		if try > 0 {
			time.Sleep(time.Duration(sleepMs) * time.Millisecond)
			sleepMs *= w.cfg.Worker.RetrySleepMultiplier
		}

		body, err := w.fetchURL(ctx, href, cian.BuildSearchURL(w.cfg.Search, region, 1))
		if err != nil {
			lastErr = err
			metrics.FetchErrors.Inc()
			w.logger.Warn("flat fetch error", zap.String("href", href), zap.Int("try", try), zap.Error(err))
			continue
		}
		if strings.Contains(body, "rate_limit") {
			lastErr = fmt.Errorf("rate limited")
			w.logger.Warn("rate limited by CIAN", zap.String("href", href), zap.Int("try", try))
			continue
		}

		info, err := cian.ParseFlatPage(body, href, region, w.logger)
		if err != nil {
			lastErr = err
			w.logger.Warn("flat parse error", zap.String("href", href), zap.Int("try", try), zap.Error(err))
			continue
		}
		return info, nil
	}

	return nil, fmt.Errorf("all %d retries failed for %s: %w", w.cfg.Worker.MaxRetries, href, lastErr)
}

func (w *Worker) fetchURL(ctx context.Context, targetURL, referer string) (string, error) {
	req, err := fhttp.NewRequestWithContext(ctx, fhttp.MethodGet, targetURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7")
	req.Header.Set("Accept-Language", "ru,en;q=0.9")
	req.Header.Set("Cache-Control", "max-age=0")
	req.Header.Set("Priority", "u=0, i")
	req.Header.Set("Sec-Ch-Ua", `"Not(A:Brand";v="8", "Chromium";v="144", "YaBrowser";v="26.3", "Yowser";v="2.5"`)
	req.Header.Set("Sec-Ch-Ua-Mobile", "?0")
	req.Header.Set("Sec-Ch-Ua-Platform", `"macOS"`)
	req.Header.Set("Sec-Fetch-Dest", "document")
	req.Header.Set("Sec-Fetch-Mode", "navigate")
	if referer != "" {
		req.Header.Set("Referer", referer)
		req.Header.Set("Sec-Fetch-Site", "same-origin")
	} else {
		req.Header.Set("Sec-Fetch-Site", "none")
	}
	req.Header.Set("Sec-Fetch-User", "?1")
	req.Header.Set("Upgrade-Insecure-Requests", "1")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/144.0.0.0 YaBrowser/26.3.0.0 Safari/537.36")

	resp, err := w.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != fhttp.StatusOK {
		return "", fmt.Errorf("HTTP %d from %s", resp.StatusCode, targetURL)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
