package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/asmisnik/realty-parser/internal/config"
	"github.com/asmisnik/realty-parser/internal/db"
	"github.com/asmisnik/realty-parser/internal/domclick"
)

func main() {
	path := flag.String("config", "/app/config.yaml", "realty-parser configuration file")
	region := flag.Int("region", 1, "RealtyRanker region ID")
	deal := flag.String("deal", "", "rent or sale")
	flag.Parse()
	if (*deal != "rent" && *deal != "sale") || *region <= 0 || flag.NArg() != 0 {
		flag.Usage()
		os.Exit(2)
	}
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, os.Interrupt)
	defer cancel()
	if err := run(ctx, *path, *region, *deal); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context, path string, region int, deal string) error {
	cfg, err := config.Load(path)
	if err != nil {
		return fmt.Errorf("reading parser configuration: %w", err)
	}
	page, err := domclick.ParseSearch(os.Stdin, cfg.Search, region, deal, 0)
	if err != nil {
		return err
	}
	database, err := db.New(ctx, cfg.Database.DSN)
	if err != nil {
		return err
	}
	defer database.Close()
	inserted := 0
	for _, flat := range page.Flats {
		if inserted >= cfg.Search.OverallLimit {
			break
		}
		added, err := database.InsertFlatAndEnqueue(ctx, flat)
		if err != nil {
			return fmt.Errorf("importing %s: %w", flat.Link, err)
		}
		if added {
			inserted++
		}
	}
	fmt.Printf("Domclick %s: %d eligible, %d inserted\n", deal, len(page.Flats), inserted)
	return nil
}
