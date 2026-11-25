package yandex

import (
	"errors"
	"fmt"
	"io"
	"net/url"
	"strconv"
	"strings"

	"golang.org/x/net/html"
)

const realtyOrigin = "https://realty.yandex.ru"

type SearchPage struct {
	ListingURLs []string `json:"listing_urls"`
	NextURL     string   `json:"next_url,omitempty"`
}

type SavedPage struct {
	Search  *SearchPage `json:"search,omitempty"`
	Listing *Listing    `json:"listing,omitempty"`
}

// ParseSavedPage reads a public search or listing page saved as HTML. It makes no network requests.
func ParseSavedPage(r io.Reader, region int) (SavedPage, error) {
	doc, err := html.Parse(r)
	if err != nil {
		return SavedPage{}, fmt.Errorf("reading Yandex Realty HTML: %w", err)
	}
	main := firstTag(doc, "main")
	if summary := firstClass(main, "OfferCard__offerCardSummary"); summary != nil {
		listing, err := parseListingHTML(doc, main, summary, region)
		return SavedPage{Listing: listing}, err
	}
	if search := parseSearchHTML(doc); len(search.ListingURLs) > 0 {
		return SavedPage{Search: &search}, nil
	}
	for n := range doc.Descendants() {
		if (n.Data == "form" && strings.Contains(attribute(n, "action"), "showcaptcha")) ||
			(n.Data == "title" && isChallengeTitle(nodeText(n))) {
			return SavedPage{}, ErrBlocked
		}
	}
	return SavedPage{}, errors.New("saved HTML has no recognizable Yandex Realty search results or listing")
}

func parseSearchHTML(doc *html.Node) SearchPage {
	var result SearchPage
	seen := make(map[string]bool)
	for n := range doc.Descendants() {
		if n.Type != html.ElementNode || n.Data != "a" {
			continue
		}
		href := attribute(n, "href")
		if attribute(n, "aria-label") == "Следующая страница" {
			if next := realtyURL(href); next != nil {
				if page, err := strconv.Atoi(next.Query().Get("page")); err == nil && page > 1 {
					result.NextURL = next.String()
				}
			}
		}
		if !hasClass(n.Parent, "OffersSerpItem__generalInfoInnerContainer") {
			continue
		}
		link := listingURL(href)
		if link != "" && !seen[link] {
			seen[link] = true
			result.ListingURLs = append(result.ListingURLs, link)
		}
	}
	return result
}

func realtyURL(value string) *url.URL {
	base, _ := url.Parse(realtyOrigin)
	u, err := base.Parse(value)
	if err != nil || u.Scheme != "https" || u.Host != base.Host || u.User != nil {
		return nil
	}
	u.Fragment = ""
	return u
}

func listingURL(value string) string {
	u := realtyURL(value)
	if u == nil {
		return ""
	}
	path, ok := strings.CutPrefix(u.Path, "/offer/")
	id := strings.TrimSuffix(path, "/")
	if _, err := strconv.ParseUint(id, 10, 64); !ok || err != nil {
		return ""
	}
	return realtyOrigin + "/offer/" + id + "/"
}

func isChallengeTitle(title string) bool {
	title = strings.ToLower(title)
	return strings.Contains(title, "вы не робот") || strings.Contains(title, "smartcaptcha") ||
		strings.Contains(title, "just a moment") || strings.Contains(title, "are you not a robot")
}

func attribute(n *html.Node, key string) string {
	if n != nil {
		for _, a := range n.Attr {
			if a.Key == key {
				return a.Val
			}
		}
	}
	return ""
}

func hasClass(n *html.Node, name string) bool {
	for _, class := range strings.Fields(attribute(n, "class")) {
		if class == name || strings.HasPrefix(class, name+"--") {
			return true
		}
	}
	return false
}

func firstTag(root *html.Node, tag string) *html.Node {
	if root != nil {
		for n := range root.Descendants() {
			if n.Type == html.ElementNode && n.Data == tag {
				return n
			}
		}
	}
	return nil
}

func classNodes(root *html.Node, class string) []*html.Node {
	var nodes []*html.Node
	if root != nil {
		for n := range root.Descendants() {
			if hasClass(n, class) {
				nodes = append(nodes, n)
			}
		}
	}
	return nodes
}

func firstClass(root *html.Node, class string) *html.Node {
	if root != nil {
		for n := range root.Descendants() {
			if hasClass(n, class) {
				return n
			}
		}
	}
	return nil
}

func nodeText(root *html.Node) string {
	if root == nil {
		return ""
	}
	var b strings.Builder
	for n := range root.Descendants() {
		if n.Type == html.TextNode {
			b.WriteString(n.Data)
		} else if n.Type == html.ElementNode && n.Data == "br" {
			b.WriteByte('\n')
		}
	}
	return strings.Join(strings.Fields(b.String()), " ")
}

func textsByClass(root *html.Node, class string) []string {
	var values []string
	for _, n := range classNodes(root, class) {
		if value := nodeText(n); value != "" {
			values = append(values, value)
		}
	}
	return values
}
