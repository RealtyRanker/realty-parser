package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/asmisnik/realty-parser/internal/db"
	"github.com/asmisnik/realty-parser/internal/yandex"
)

type report struct {
	SearchPages []yandex.SearchPage `json:"search_pages,omitempty"`
	Listings    []*yandex.Listing   `json:"listings,omitempty"`
	CatalogNew  *int                `json:"new_catalog_rows,omitempty"`
}

func main() {
	flag.Usage = func() {
		fmt.Fprintln(flag.CommandLine.Output(), "Usage: yandex-import -region ID [-db FILE] (page.html ... | -stdin-html-batch)")
		flag.PrintDefaults()
	}
	region := flag.Int("region", 0, "RealtyRanker region ID (1 for Moscow)")
	database := flag.String("db", "", "existing RealtyRanker SQLite file for catalog import")
	batch := flag.Bool("stdin-html-batch", false, "read a JSON array of HTML pages from standard input")
	flag.Parse()
	if *region <= 0 || (flag.NArg() == 0) != *batch {
		flag.Usage()
		os.Exit(2)
	}
	if err := run(*region, *database, *batch, flag.Args()); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(region int, database string, batch bool, paths []string) error {
	var output report
	seen := make(map[string]bool)
	parse := func(r io.Reader) error {
		page, err := yandex.ParseSavedPage(r, region)
		if err != nil {
			return err
		}
		if page.Search != nil {
			output.SearchPages = append(output.SearchPages, *page.Search)
		}
		if page.Listing != nil && !seen[page.Listing.Flat.Link] {
			output.Listings = append(output.Listings, page.Listing)
			seen[page.Listing.Flat.Link] = true
		}
		return nil
	}
	if batch {
		var pages []string
		if err := json.NewDecoder(os.Stdin).Decode(&pages); err != nil {
			return fmt.Errorf("reading HTML batch: %w", err)
		}
		for i, page := range pages {
			if err := parse(strings.NewReader(page)); err != nil {
				return fmt.Errorf("page %d: %w", i+1, err)
			}
		}
	}
	for _, path := range paths {
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		err = parse(file)
		file.Close()
		if err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
	}
	if database != "" {
		newRows, err := store(context.Background(), database, output.Listings)
		if err != nil {
			return err
		}
		output.CatalogNew = &newRows
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(output)
}

func store(ctx context.Context, path string, listings []*yandex.Listing) (int, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return 0, err
	}
	dsn := (&url.URL{Scheme: "file", Path: absolute, RawQuery: "mode=rw"}).String()
	catalog, err := db.New(ctx, dsn)
	if err != nil {
		return 0, err
	}
	defer catalog.Close()
	count := 0
	for _, listing := range listings {
		inserted, err := catalog.InsertCatalogFlat(ctx, listing.Flat)
		if err != nil {
			return count, fmt.Errorf("storing %s: %w", listing.Flat.Link, err)
		}
		if inserted {
			count++
		}
	}
	return count, nil
}
