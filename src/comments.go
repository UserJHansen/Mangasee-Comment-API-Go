package main

import (
	"math"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

func commentResponse(c *gin.Context) {
	page := c.Param("number")
	max := uint32(math.Ceil(float64(len(comments))/1000) - 1)

	pageNo, err := conv[uint32](page)
	if err != nil || pageNo > max {
		c.JSON(400, &Result[string]{Status: "Bad Request", Result: "Invalid page"})
		return
	}

	filtered := comments[pageNo*1000 : uint32(math.Min(float64(pageNo*1000+1000), float64(len(comments))))]

	c.JSON(200, &Result[gin.H]{Status: "OK", Result: gin.H{"page": filtered, "max": max}})
}

func since(c *gin.Context, comments []Comment) ([]Comment, error) {
	filtered := []Comment{}
	since, err := time.Parse(time.RFC3339, c.Param("since"))
	if err != nil {
		c.JSON(400, &Result[string]{Status: "Bad Request", Result: "Invalid since"})
		return nil, err
	}
	for _, comment := range comments {
		if comment.Timestamp.After(since) {
			filtered = append(filtered, comment)
		} else {
			for _, reply := range comment.Replies {
				if reply.Timestamp.After(since) {
					filtered = append(filtered, comment)
					break
				}
			}
		}
	}
	return filtered, nil
}

func sinceResponse(c *gin.Context) {
	filtered, err := since(c, comments)
	if err != nil {
		return
	}

	c.JSON(200, &Result[[]Comment]{Status: "OK", Result: filtered})
}

func mangaResponse(c *gin.Context) {
	filtered := []Comment{}
	for _, comment := range comments {
		if strings.ToLower(comment.MangaName) == c.Param("manga") {
			filtered = append(filtered, comment)
		}
	}

	c.JSON(200, &Result[[]Comment]{Status: "OK", Result: filtered})
}

func mangaSinceResponse(c *gin.Context) {
	filtered := []Comment{}
	for _, comment := range comments {
		if strings.ToLower(comment.MangaName) == c.Param("manga") {
			filtered = append(filtered, comment)
		}
	}
	filtered, err := since(c, filtered)
	if err != nil {
		return
	}

	c.JSON(200, &Result[[]Comment]{Status: "OK", Result: filtered})
}

func discussionCommentsResponse(c *gin.Context) {
	id, err := conv[uint32](c.Param("id"))

	if err != nil {
		c.JSON(400, &Result[string]{Status: "Bad Request", Result: "Invalid id"})
		return
	}

	if id < 10000 {
		for _, bid := range discussionIds {
			if (bid-id)%10000 == 0 {
				id = bid
				break
			}
		}
	}

	filtered := []Comment{}
	for _, comment := range comments {
		if comment.DiscussionID == id {
			filtered = append(filtered, comment)
		}
	}

	c.JSON(200, &Result[[]Comment]{Status: "OK", Result: filtered})
}

func discussionCommentsSinceResponse(c *gin.Context) {
	id, err := conv[uint32](c.Param("id"))

	if err != nil {
		c.JSON(400, &Result[string]{Status: "Bad Request", Result: "Invalid id"})
		return
	}

	if id < 10000 {
		for _, bid := range discussionIds {
			if (bid-id)%10000 == 0 {
				id = bid
				break
			}
		}
	}

	filtered := []Comment{}
	for _, comment := range comments {
		if comment.DiscussionID == id {
			filtered = append(filtered, comment)
		}
	}
	filtered, err = since(c, filtered)
	if err != nil {
		return
	}

	c.JSON(200, &Result[[]Comment]{Status: "OK", Result: filtered})
}

func inLastResponse(c *gin.Context) {
	duration, err := time.ParseDuration(c.Param("duration"))

	if err != nil {
		c.JSON(400, &Result[string]{Status: "Bad Request", Result: "Invalid duration"})
		return
	}

	filtered := []Comment{}
	for _, comment := range comments {
		if time.Since(comment.Timestamp) < duration {
			filtered = append(filtered, comment)
		} else {
			for _, reply := range comment.Replies {
				if time.Since(reply.Timestamp) < duration {
					filtered = append(filtered, comment)
					break
				}
			}
		}
	}

	c.JSON(200, &Result[[]Comment]{Status: "OK", Result: filtered})
}

func userCommentsResponse(c *gin.Context) {
	user := c.Param("user")
	id, err := conv[uint32](user)
	if err != nil {
		for _, mapUser := range userMap {
			if strings.EqualFold(mapUser.Name, user) {
				id = mapUser.ID
				break
			}
		}
	}

	filtered := []Comment{}
	for _, comment := range comments {
		if comment.UserID == id {
			filtered = append(filtered, comment)
		} else {
			for _, reply := range comment.Replies {
				if reply.UserID == id {
					filtered = append(filtered, comment)
					break
				}
			}
		}
	}

	c.JSON(200, &Result[[]Comment]{Status: "OK", Result: filtered})
}
