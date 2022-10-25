package main

import (
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	ginprometheus "github.com/zsais/go-gin-prometheus"
)

var (
	// Prom counters
	commentNo = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "comments_total",
			Help: "Total number of Comments.",
		})
	commentCounterVal = 0.0
	replyNo           = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "reply_total",
			Help: "Total number of Replies.",
		})
	replyCounterVal = 0.0
	userNo          = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "user_total",
			Help: "Total number of Users cached.",
		})
	userCounterVal     = 0.0
	totalPossibleUsers = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "total_possible_users",
			Help: "Total number of Users possible based on the maximum ID.",
		})
	possibleUserVal = 0.0
	scanTime        = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "scan_time",
			Help: "Time taken to scan for new manga/comments in milliseconds",
		}, []string{"type"})
	numManga = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "number_manga",
			Help: "Number of manga in cache ",
		})
	numErrors = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "errors_total",
			Help: "Total number of errors encountered.",
		})
)

func getProm(ginInstance *gin.Engine) {
	if *withProm {
		p := ginprometheus.NewPrometheus("mangasee")
		p.ReqCntURLLabelMappingFn = func(c *gin.Context) string {
			url := c.Request.URL.Path
			for _, p := range c.Params {
				url = strings.Replace(url, p.Value, fmt.Sprintf(":%s", p.Key), 1)
			}
			return url
		}
		p.Use(ginInstance)

		// Update counters
		commentNo.Add(float64(len(comments)))
		commentCounterVal += float64(len(comments))
		for _, comment := range comments {
			replyNo.Add(float64(len(comment.Replies)))
			replyCounterVal += float64(len(comment.Replies))
		}

		max := uint32(0)
		for _, user := range userMap {
			if user.ID > max {
				max = user.ID
			}
		}

		userNo.Add(float64(len(userMap)))
		userCounterVal = float64(len(userMap))

		totalPossibleUsers.Add(float64(max))
		possibleUserVal = float64(max)
	}
}
