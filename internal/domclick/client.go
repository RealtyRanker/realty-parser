package domclick

import (
	"context"
	"encoding/json"
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

const searchEndpoint = "https://bff-search-web.domclick.ru/api/offers/v1"
const pageSize = 20
const maxResponseBytes = 8 << 20

var ErrBlocked = errors.New("Domclick declined search access")

type Client struct {
	http *http.Client
	url  string
}

func New(timeout time.Duration) *Client {
	return &Client{
		url: searchEndpoint,
		http: &http.Client{
			Timeout: timeout,
			CheckRedirect: func(*http.Request, []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}
}

type SearchPage struct {
	Flats   []*model.FlatInfo
	HasMore bool
}

func (c *Client) Search(ctx context.Context, filters config.SearchConfig, region int, location config.DomclickRegion, dealType string, page int) (SearchPage, error) {
	if dealType != "rent" && dealType != "sale" {
		return SearchPage{}, fmt.Errorf("unsupported Domclick deal type %q", dealType)
	}
	if page < 1 || location.Address == "" || location.AID <= 0 {
		return SearchPage{}, errors.New("invalid Domclick page or region")
	}
	target, err := url.Parse(c.url)
	if err != nil {
		return SearchPage{}, err
	}
	offset := (page - 1) * pageSize
	query := target.Query()
	query.Set("address", location.Address)
	query.Set("aids", strconv.Itoa(location.AID))
	query.Set("offset", strconv.Itoa(offset))
	query.Set("limit", strconv.Itoa(pageSize))
	query.Set("sort", "published")
	query.Set("sort_dir", "desc")
	query.Set("deal_type", dealType)
	query.Set("category", "living")
	query.Set("offer_type", "flat")
	if dealType == "sale" {
		query.Add("offer_type", "layout")
	}
	target.RawQuery = query.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target.String(), nil)
	if err != nil {
		return SearchPage{}, err
	}
	req.Header.Set("Accept", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return SearchPage{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden ||
		resp.StatusCode == http.StatusTooManyRequests || (resp.StatusCode >= 300 && resp.StatusCode < 400) {
		return SearchPage{}, fmt.Errorf("%w (HTTP %d)", ErrBlocked, resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		return SearchPage{}, fmt.Errorf("Domclick search returned HTTP %d", resp.StatusCode)
	}
	return ParseSearch(resp.Body, filters, region, dealType, offset)
}

func ParseSearch(r io.Reader, filters config.SearchConfig, region int, dealType string, offset int) (SearchPage, error) {
	var envelope struct {
		Result struct {
			Items      []json.RawMessage
			Pagination struct{ Offset, Limit, Total int }
		}
	}
	if err := json.NewDecoder(io.LimitReader(r, maxResponseBytes)).Decode(&envelope); err != nil {
		return SearchPage{}, fmt.Errorf("decoding Domclick search: %w", err)
	}
	items := envelope.Result.Items
	if items == nil {
		return SearchPage{}, errors.New("Domclick search response has no items array")
	}
	result := SearchPage{Flats: make([]*model.FlatInfo, 0, len(items))}
	seen := make(map[string]bool)
	recognized := 0
	for _, raw := range items {
		var listing offer
		if err := json.Unmarshal(raw, &listing); err != nil || listing.ID <= 0 {
			continue
		}
		recognized++
		if f := listing.flat(region, dealType); f != nil && !seen[f.Link] && filters.MatchesFlat(f) {
			result.Flats = append(result.Flats, f)
			seen[f.Link] = true
		}
	}
	if len(items) > 0 && recognized == 0 {
		return SearchPage{}, errors.New("Domclick search response has no recognizable listings")
	}
	pagination := envelope.Result.Pagination
	result.HasMore = len(items) > 0 && pagination.Offset == offset && offset+pageSize < pagination.Total
	return result, nil
}
