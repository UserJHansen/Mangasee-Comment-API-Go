package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"sync"
	"time"
)

type SearchResponse struct {
	IndexName      string   `json:"i"`
	StringName     string   `json:"s"`
	AlternateNames []string `json:"a"`
}
type Response[T any] struct {
	Success bool `json:"success"`
	Val     T    `json:"val"`
}
type RawReply struct {
	CommentID      string
	UserID         string
	Username       string
	CommentContent string
	TimeCommented  string
}
type RawComment struct {
	CommentID      string
	UserID         string
	Username       string
	CommentContent string
	TimeCommented  string
	ReplyCount     string
	LikeCount      string
	Liked          bool
	ShowReply      bool
	Replying       bool
	ReplyLimit     int16
	ReplyMessage   string
	Replies        []RawReply
}

type CommentResponse Response[[]RawComment]
type ReplyResponse Response[[]RawReply]

func conv[T uint | int | uint8 | uint16 | uint32 | uint64 | int8 | int16 | int32 | int64](str string) T {
	val, _ := strconv.Atoi(str)
	return T(val)
}

func getMangaReplies(comment *RawComment, name string) error {
	postBody, _ := json.Marshal(map[string]string{
		"TargetID": comment.CommentID,
	})
	responseBody := bytes.NewBuffer(postBody)
	//Leverage Go's HTTP Post function to make request
	resp, err := http.Post(*server+"manga/reply.get.php", "application/json", responseBody)

	if err != nil {
		fmt.Printf("[COMMENT-CACHE] Error reading replies for: %s, commentID: %s, error: %s\n", name, comment.CommentID, err)
		return err
	}

	defer resp.Body.Close()
	var replies ReplyResponse
	if err := json.NewDecoder(resp.Body).Decode(&replies); err != nil {
		fmt.Printf("[COMMENT-CACHE] Error parsing json for: %s, commentID: %s, error: %s\n", name, comment.CommentID, err)
		return err
	}
	if !replies.Success {
		fmt.Println("[COMMENT-CACHE] Status code is bad for replies:", name, "value:", replies)
		return err
	}
	if *verbose {
		fmt.Println("[COMMENT-CACHE] Successfully scanned", name, "commentID:", comment.CommentID, "replies:", len(replies.Val))
	}
	comment.Replies = replies.Val
	return nil
}

func scanManga(manga SearchResponse) ([]RawComment, error) {
	values := map[string]string{"IndexName": manga.IndexName}
	json_data, _ := json.Marshal(values)

	resp, err := http.Post(*server+"manga/comment.get.php", "application/json",
		bytes.NewBuffer(json_data))

	if err != nil {
		fmt.Printf("[COMMENT-CACHE] Error reading comments for: %s, error: %s\n", manga.IndexName, err)
		return nil, err
	}

	defer resp.Body.Close()
	res, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("[COMMENT-CACHE] Error reading comments for: %s, error: %s\n", manga.IndexName, err)
		return nil, err
	}

	var response CommentResponse
	if err := json.Unmarshal(res, &response); err != nil {
		fmt.Printf("[COMMENT-CACHE] Error decoding json for manga %s, error: %s\n", manga.IndexName, err)
		return nil, err
	}
	if !response.Success {
		fmt.Println("[COMMENT-CACHE] Status code is bad for manga:", manga.IndexName, "value:", response)
		return nil, fmt.Errorf("status code is bad for manga: %s, result: %s", manga.IndexName, res)
	}
	return response.Val, nil
}

func scanAllManga() error {
	start := time.Now().UnixMicro()
	fmt.Println("[COMMENT-CACHE] Starting scan...")
	resp, err := http.Get(*server + "_search.php")
	if err != nil {
		fmt.Println("[COMMENT-CACHE] Error getting Manga: ", err)
		return err
	}
	defer resp.Body.Close()

	var mangas []SearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&mangas); err != nil {
		fmt.Println("[COMMENT-CACHE] Error decoding Manga: ", err)
		return err
	}

	var wg sync.WaitGroup
	guard := make(chan struct{}, *procs)
	wg.Add(len(mangas))
	commentResults := make([][]RawComment, len(mangas))
	for i, manga := range mangas {
		guard <- struct{}{}
		go func(manga SearchResponse, i int) {
			result, err := scanManga(manga)
			if err == nil {
				if *verbose {
					fmt.Println("[COMMENT-CACHE] Successfully scanned", manga.IndexName, fmt.Sprint(i)+"/"+fmt.Sprint(len(mangas)))
				}
				commentResults[i] = result
			}
			<-guard
			wg.Done()
		}(manga, i)
	}

	fmt.Println("[COMMENT-CACHE] Took", (time.Now().UnixMicro()-start)/time.Second.Microseconds(), "seconds to scan", len(mangas), "mangas")
	fmt.Println("[COMMENT-CACHE] That's an average of", (time.Now().UnixMicro()-start)/int64(len(mangas)), "microseconds per manga")

	scanTime.Set(float64((time.Now().UnixMicro() - start)/time.Millisecond.Microseconds()))
	numManga.Set(float64(len(mangas)))

	start = time.Now().UnixMicro()
	numberRequests := 0
	// Get replies for each comment
	for i, manga := range commentResults {
		for _, comment := range manga {
			if int16(len(comment.Replies)) > comment.ReplyLimit {
				wg.Add(len(mangas))
				guard <- struct{}{}
				go func(name string, comment *RawComment) {
					_ = getMangaReplies(comment, name)
					numberRequests++
					<-guard
					wg.Done()
				}(mangas[i].IndexName, &comment)
			}
		}
	}

	fmt.Println("[COMMENT-CACHE] Took", (time.Now().UnixMicro()-start)/time.Second.Microseconds(), "seconds to get replies for", numberRequests, "comments")
	fmt.Println("[COMMENT-CACHE] That's an average of", (time.Now().UnixMicro()-start)/int64(numberRequests), "microseconds per reply")
	// Extract number of comments and replies
	// and create a rough map of UserIDs to usernames
	start = time.Now().UnixMicro()
	newMap := []Username{}
	for _, manga := range commentResults {
		for _, comment := range manga {
			for _, reply := range comment.Replies {
				newMap = append(newMap, Username{
					ID:   conv[uint32](reply.UserID),
					Name: reply.Username,
				})
			}
			newMap = append(newMap, Username{
				ID:   conv[uint32](comment.UserID),
				Name: comment.Username,
			})
		}
	}

	fmt.Println("[COMMENT-CACHE] Took", (time.Now().UnixMicro() - start)/time.Millisecond.Microseconds(), "milliseconds to extract usernames")

	start = time.Now().UnixMicro()
	// Deduplicate the map and update Prom
	newMap = append(newMap, userMap...)
	keys := make(map[int]bool)
	dedupedUsers := []Username{}
	for _, entry := range newMap {
		if _, value := keys[int(entry.ID)]; !value {
			keys[int(entry.ID)] = true
			dedupedUsers = append(dedupedUsers, entry)
		}
	}
	userMap = dedupedUsers
	userNo.Add(float64(len(userMap)) - userCounterVal)
	userCounterVal = float64(len(userMap))

	fmt.Println("[COMMENT-CACHE] Took", (time.Now().UnixMicro() - start), "microseconds to deduplicate", len(userMap), "users")

	start = time.Now().UnixMicro()
	numberProcessed := uint32(0)
	totalReplies := uint32(0)
	// Create a proper tree of comments and replies
	for i, manga := range commentResults {
		for _, comment := range manga {
			newcomment := Comment{
				ID:           conv[uint32](comment.CommentID),
				UserID:       conv[uint32](comment.UserID),
				Content:      comment.CommentContent,
				Likes:        conv[int16](comment.LikeCount),
				Timestamp:    conv[uint64](comment.TimeCommented),
				DiscussionID: 0,
				MangaName:    mangas[i].IndexName,
			}
			numberProcessed++
			totalReplies += uint32(len(comment.Replies))
			for _, reply := range comment.Replies {
				newreply := Reply{
					ID:        conv[uint32](reply.CommentID),
					UserID:    conv[uint32](reply.UserID),
					Content:   reply.CommentContent,
					Timestamp: conv[uint64](reply.TimeCommented),
				}
				newcomment.Replies = append(newcomment.Replies, newreply)
				numberProcessed++
			}
			comments = append(comments, newcomment)
		}
	}
	// Deduplicate the comments
	keys = make(map[int]bool)
	dedupedComments := []Comment{}
	for _, comment := range comments {
		if _, value := keys[int(comment.ID)]; !value {
			keys[int(comment.ID)] = true
			dedupedComments = append(dedupedComments, comment)
		}
	}
	comments = dedupedComments
	commentNo.Add(float64(len(comments)) - commentCounterVal)
	commentCounterVal = float64(len(comments))
	replyNo.Add(float64(totalReplies) - replyCounterVal)
	replyCounterVal = float64(totalReplies)

	fmt.Println("[COMMENT-CACHE] Took", (time.Now().UnixMicro()-start)/1000^3, "microseconds to create a proper tree of comments and replies")

	// Write to file
	if res := save(); res != nil {
		return res
	}

	return nil
}