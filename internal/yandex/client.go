package yandex

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/asmisnik/realty-parser/internal/config"
	"github.com/asmisnik/realty-parser/internal/model"
)

const endpoint = "https://realty.yandex.ru/gate/react-page/get/"
const maxResponseBytes = 16 << 20

var ErrBlocked = errors.New("Yandex Realty returned a challenge or blocked the request")

type Client struct {
	http *http.Client
	url  string
}

func New(timeout time.Duration) *Client {
	return &Client{url: endpoint, http: &http.Client{
		Timeout: timeout,
		CheckRedirect: func(req *http.Request, previous []*http.Request) error {
			if req.URL.Hostname() != previous[0].URL.Hostname() || req.URL.Path == "/showcaptcha" {
				return http.ErrUseLastResponse
			}
			if len(previous) >= 4 {
				return errors.New("too many Yandex Realty redirects")
			}
			return nil
		},
	}}
}

// Search reads a single response from the site's search frontend.
func (c *Client) Search(ctx context.Context, filters config.SearchConfig, region, rgid int, dealType string) ([]*model.FlatInfo, error) {
	target, err := url.Parse(c.url)
	if err != nil {
		return nil, err
	}
	query := target.Query()
	query.Set("_pageType", "search")
	query.Set("page", "SEARCH")
	query.Add("_providers", "search")
	query.Set("category", "APARTMENT")
	query.Set("rgid", strconv.Itoa(rgid))
	query.Set("sort", "DATE_DESC")
	switch dealType {
	case "sale":
		query.Set("type", "SELL")
	case "rent":
		query.Set("type", "RENT")
	default:
		return nil, fmt.Errorf("unsupported deal type %q", dealType)
	}
	for key, value := range map[string]int{
		"priceMin": filters.MinPrice, "priceMax": filters.MaxPrice,
		"areaMin": filters.MinArea, "areaMax": filters.MaxArea,
	} {
		if value > 0 {
			query.Set(key, strconv.Itoa(value))
		}
	}
	if len(filters.Rooms) > 0 {
		rooms := make(map[string]bool)
		for _, room := range filters.Rooms {
			switch {
			case room == 1:
				rooms["STUDIO"] = true
			case room >= 5:
				rooms["PLUS_4"] = true
			case room > 1:
				rooms[strconv.Itoa(room-1)] = true
			}
		}
		for _, room := range []string{"STUDIO", "1", "2", "3", "PLUS_4"} {
			if rooms[room] {
				query.Add("roomsTotal", room)
			}
		}
	}
	target.RawQuery = query.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusTooManyRequests ||
		(resp.StatusCode >= 300 && resp.StatusCode < 400) {
		return nil, fmt.Errorf("%w (HTTP %d)", ErrBlocked, resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Yandex Realty returned HTTP %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes+1))
	if err != nil {
		return nil, err
	}
	if len(body) > maxResponseBytes {
		return nil, errors.New("Yandex Realty search response exceeds size limit")
	}
	flats, err := parseSearch(body, region, dealType)
	if err != nil {
		return nil, err
	}
	result := flats[:0]
	for _, f := range flats {
		if matchesFilters(f, filters) {
			result = append(result, f)
		}
	}
	return result, nil
}

func matchesFilters(f *model.FlatInfo, cfg config.SearchConfig) bool {
	if f.Price < cfg.MinPrice || (cfg.MaxPrice > 0 && f.Price > cfg.MaxPrice) ||
		f.TotalArea < float64(cfg.MinArea) || (cfg.MaxArea > 0 && f.TotalArea > float64(cfg.MaxArea)) {
		return false
	}
	if len(cfg.Rooms) == 0 {
		return true
	}
	for _, room := range cfg.Rooms {
		if (room <= 5 && f.RoomNumber == room-1) || (room == 6 && f.RoomNumber >= 5) {
			return true
		}
	}
	return false
}
