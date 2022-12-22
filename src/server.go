package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	ratelimit "github.com/JGLTechnologies/gin-rate-limit"
	"github.com/Xiami2012/pidfile-go"
	"github.com/gin-gonic/gin"
)

// Command line flags
var saveLoc = flag.String("o", "cache.json", "location where the cache will be stored")

var withProm = flag.Bool("p", true, "Will prometheus be included")
var secure = flag.Bool("secure", false, "Whether to use https or not")
var server = flag.String("s", "https://mangasee123.com/", "Server to connect to, Mangasee or Manga4Life")

var procs = flag.Int("procs", 100, "Number of processes used for scanning")
var interval = flag.Int("i", 4, "Interval between scans in minutes")

var verbose = flag.Bool("v", false, "Verbose output")
var timing = flag.Bool("t", false, "Time the scan")
var clearcache = flag.Bool("c", false, "Clear the cache")
var ignorePID = flag.Bool("P", false, "Ignore PID file")
var pidLoc = flag.String("pid", "comment-cache.pid", "Location of the PID file")
var logLoc = flag.String("l", "comment-cache.log", "Location of the log file")

var (
	comments      = []Comment{}
	userMap       = []Username{}
	discussions   = []Discussion{}
	discussionIds = []uint32{}
	deleted 	 = []uint32{}
	out           = io.MultiWriter(os.Stdout)
)

func main() {
	// Load cli flags
	flag.Parse()

	if !*ignorePID {
		fmt.Println("Creating PID file at:", *pidLoc)
		if err := pidfile.Write(*pidLoc); err != nil {
			if errors.Is(err, pidfile.ErrPIDFileInUse) {
				Printf("Instance/Service is already running. Exiting.")
				os.Exit(1)
			}
			panic(fmt.Sprintf("Failed to create PIDFile: %s\n", err.Error()))
		}
	}

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

	// Log file
	gin.DisableConsoleColor()
	f, _ := os.Create(*logLoc)
	gin.DefaultWriter = io.MultiWriter(f, os.Stdout)
	out = io.MultiWriter(f, os.Stdout)

	// Get gin
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(gin.Logger())
	_ = r.SetTrustedProxies(nil)

	// Prom setup
	getProm(r)

	// setup rate limiting
	store := ratelimit.InMemoryStore(&ratelimit.InMemoryOptions{
		Rate:  time.Minute,
		Limit: 120,
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
	r.GET("/discussions", discussionResponse)
	r.GET("/comments/page/:number", commentResponse)
	r.GET("/comments/since/:since", sinceResponse)
	r.GET("/comments/manga/:manga", mangaResponse)
	r.GET("/comments/manga/:manga/since/:since", mangaSinceResponse)
	r.GET("/comments/discussion/:id", discussionCommentsResponse)
	r.GET("/comments/discussion/:id/since/:since", discussionCommentsSinceResponse)
	r.GET("/comments/inlast/:duration", inLastResponse)
	r.GET("/comments/from/:user", userCommentsResponse)
	r.NoRoute(fourofour)

	spawnScanner()

	// Graceful shutdown
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-shutdown
		_ = pidfile.Remove(*pidLoc)
		if err := save(); err != nil {
			log.Fatal("Failed to save:", err)
		}
		log.Println("Closing")
		os.Exit(0)
	}()

	if (*secure) {
		setupTls(r)
	} else {
		log.Fatal(r.Run(":8080"))
	}
}
