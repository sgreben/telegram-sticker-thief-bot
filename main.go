package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"time"

	telegram "github.com/sgreben/telegram-sticker-thief-bot/internal/telebot.v2"
)

var config struct {
	Token        string
	Timeout      time.Duration
	RateLimit    time.Duration
	MaxRetries   int
	DefaultEmoji string
	Verbose      bool
}

var appName = "telegram-sticker-thief-bot"
var version = "dev"
var jsonOut = json.NewEncoder(os.Stdout)

func init() {
	log.SetOutput(os.Stderr)
	log.SetFlags(0)
	log.SetPrefix(fmt.Sprintf("[%s %s] ", filepath.Base(appName), version))

	config.Timeout = 2 * time.Second
	config.RateLimit = time.Second / 10
	config.DefaultEmoji = "⭐️"
	config.MaxRetries = 32

	flag.DurationVar(&config.Timeout, "timeout", config.Timeout, "")
	flag.DurationVar(&config.RateLimit, "rate-limit", config.RateLimit, "")
	flag.StringVar(&config.Token, "token", config.Token, "")
	flag.BoolVar(&config.Verbose, "verbose", config.Verbose, "")
	flag.BoolVar(&config.Verbose, "v", config.Verbose, "(alas for -verbose)")
	flag.Parse()

	if !config.Verbose {
		jsonOut = json.NewEncoder(ioutil.Discard)
	}
}

func main() {
	botAPI, err := telegram.NewBot(telegram.Settings{
		Token:  config.Token,
		Poller: &telegram.LongPoller{Timeout: config.Timeout},
	})
	if err != nil {
		log.Fatal(err)
		return
	}
	bot := &stickerThiefBot{
		Bot:  botAPI,
		Tick: time.Tick(config.RateLimit),
	}
	bot.init()
	bot.Start()
}
