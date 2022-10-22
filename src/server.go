package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	ratelimit "github.com/JGLTechnologies/gin-rate-limit"
	"github.com/gin-gonic/gin"
)

// Command line flags
var saveLoc = flag.String("o", "cache.json", "location where the cache will be stored")
var withProm = flag.Bool("p", true, "Will prometheus be included")
var bindAddr = flag.String("b", ":9500", "Which address to bind to")
var server = flag.String("s", "https://mangasee123.com/", "Server to connect to, Mangasee or Manga4Life")
var procs = flag.Int("procs", 100, "Number of processes used for scanning")
var interval = flag.Int("i", 2, "Interval between scans in minutes")

var verbose = flag.Bool("v", false, "Verbose output")
var timing = flag.Bool("t", false, "Time the scan")
var clearcache = flag.Bool("c", false, "Clear the cache")

var (
	comments      = []Comment{}
	userMap       = []Username{}
	discussions   = []Discussion{}
	discussionIds = []uint32{}
)

func main() {
	// Load cli flags
	flag.Parse()

	// Load from cache file
	if !*clearcache {
		if err := load(); err != nil {
			log.Fatal(err)
		}
	}

	// Make sure that we can save
	if err := save(); err != nil {
		log.Fatal("Failed to save:", err)
	}

	// Get gin
	r := gin.Default()

	// Prom setup
	getProm(r)

	// setup rate limiting
	store := ratelimit.InMemoryStore(&ratelimit.InMemoryOptions{
		Rate:  time.Second,
		Limit: 1,
	})
	ratelimit := ratelimit.RateLimiter(store, &ratelimit.Options{
		ErrorHandler: errorHandler,
		KeyFunc: func(c *gin.Context) string {
			return c.ClientIP()
		},
	})
	r.Use(ratelimit)

	// Routes
	r.GET("/users", userResponse)
	r.NoRoute(fourofour)

	spawnScanner()

	// Graceful shutdown
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-shutdown
		if err := save(); err != nil {
			log.Fatal("Failed to save:", err)
		}
		log.Println("Closing")
		os.Exit(0)
	}()

	log.Fatal(r.Run(*bindAddr))
}
