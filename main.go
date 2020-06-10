package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"kaepora/internal/back"
	"kaepora/internal/bot"
	"kaepora/internal/generator/oot"
	"kaepora/internal/generator/oot/settings"
	"kaepora/internal/global"
	"kaepora/internal/web"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"

	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	log.SetFlags(0) // we syslog in prod so we don't care about time here
	flag.Parse()

	switch flag.Arg(0) { // commands not requiring a back
	case "version":
		fmt.Fprintf(os.Stdout, "Kaepora %s\n", global.Version)
		return
	case "help":
		fmt.Fprint(os.Stdout, help())
		return
	case "settings":
		if err := generateSettingsStats(); err != nil {
			log.Fatal(err)
		}
		return
	}

	log.Printf("info: Starting Kaepora %s", global.Version)
	back, err := back.New(
		"sqlite3", "./kaepora.db",
		os.Getenv("KAEPORA_OOTR_API_KEY"),
	)
	if err != nil {
		log.Fatal(err)
	}

	switch flag.Arg(0) {
	case "fixtures":
		if err := back.LoadFixtures(); err != nil {
			log.Fatal(err)
		}
	case "serve":
		if err := serve(back); err != nil {
			log.Fatal(err)
		}
	case "rerank":
		if err := back.Rerank(flag.Arg(1)); err != nil {
			log.Fatal(err)
		}
	default:
		fmt.Fprint(os.Stderr, help())
		os.Exit(1)
	}
}

func help() string {
	return fmt.Sprintf(`
Kaepora is a tool to manage the "Ocarina of Time: Randomizer"
competitive ladder.

Usage: %[1]s COMMAND [ARGS…]

COMMANDS
    fixtures    create default data for quick testing during development
    help        display this help
    serve       start the Discord bot
    settings    output settings randomizer stats
    version     display the current version

    rerank SHORTCODE  recompute all rankings in a league
`,
		os.Args[0],
	)
}

func serve(b *back.Back) error {
	done := make(chan struct{})
	signaled := make(chan os.Signal, 1)
	signal.Notify(signaled, syscall.SIGINT, syscall.SIGTERM)

	bot, err := bot.New(b, os.Getenv("KAEPORA_DISCORD_TOKEN"))
	if err != nil {
		return err
	}

	server, err := web.NewServer(b, os.Getenv("KAEPORA_WEB_TOKEN_KEY"))
	if err != nil {
		return err
	}

	var wg sync.WaitGroup
	go b.Run(&wg, done)
	go bot.Serve(&wg, done)
	go server.Serve(&wg, done)

	sig := <-signaled
	log.Printf("warning: received signal %d", sig)
	close(done)

	log.Print("info: waiting for complete shutdown")
	wg.Wait()
	log.Print("info: shutdown complete")

	return nil
}

func generateSettingsStats() error {
	baseDir, err := oot.GetBaseDir()
	if err != nil {
		return err
	}

	s, err := settings.Load(filepath.Join(baseDir, settings.DefaultName))
	if err != nil {
		return err
	}

	max := 102400
	count := map[string]map[string]int{} // name => value => count
	for i := 0; i < max; i++ {
		settings := s.Shuffle(uuid.New().String(), oot.SettingsCostBudget)

		for name, value := range settings {
			if _, ok := count[name]; !ok {
				count[name] = map[string]int{}
			}

			count[name][fmt.Sprintf("%v", value)]++
		}
	}

	stats := web.NamedPct2DFrom2DMap(count, max)
	w := csv.NewWriter(os.Stdout)
	for _, v := range stats {
		_ = w.Write([]string{v.Name, v.Value, fmt.Sprintf("%.2f", v.Pct)})
	}
	w.Flush()

	return nil
}
