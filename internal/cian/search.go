package cian

import (
	"fmt"
	"strings"

	"github.com/asmisnik/realty-parser/internal/config"
)

const baseSearchURL = "https://www.cian.ru/cat.php?engine_version=2&offer_type=flat"

// Deal types accepted by BuildSearchURL, matching session.DealType* in subscription-handler.
const (
	DealTypeRent = "rent"
	DealTypeSale = "sale"
)

// BuildSearchURL constructs a CIAN search URL from the given filter config, region id,
// deal type ("rent" or "sale") and page number.
func BuildSearchURL(cfg config.SearchConfig, region int, dealType string, page int) string {
	var sb strings.Builder
	sb.WriteString(baseSearchURL)
	sb.WriteString("&deal_type=" + dealType)
	if dealType == DealTypeRent {
		// type=4 selects long-term rent (as opposed to daily/short-term).
		sb.WriteString("&type=4")
	}
	sb.WriteString(fmt.Sprintf("&region=%d", region))
	addIntParam(&sb, "minprice", cfg.MinPrice)
	addIntParam(&sb, "maxprice", cfg.MaxPrice)
	addIntParam(&sb, "minarea", cfg.MinArea)
	addIntParam(&sb, "maxarea", cfg.MaxArea)
	if cfg.Sort != "" {
		sb.WriteString("&sort=" + cfg.Sort)
	}
	for _, room := range cfg.Rooms {
		sb.WriteString(fmt.Sprintf("&room%d=1", room))
	}
	addIntParam(&sb, "limit", cfg.Limit)
	addIntParam(&sb, "p", page)
	return sb.String()
}

func addIntParam(sb *strings.Builder, name string, value int) {
	if value != 0 {
		sb.WriteString(fmt.Sprintf("&%s=%d", name, value))
	}
}
