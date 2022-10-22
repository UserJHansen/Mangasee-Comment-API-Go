package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

func getReplies(comment *RawComment, url string) error {
	postBody, _ := json.Marshal(map[string]string{
		"TargetID": comment.CommentID,
	})
	responseBody := bytes.NewBuffer(postBody)
	//Leverage Go's HTTP Post function to make request
	resp, err := http.Post(*server+url, "application/json", responseBody)

	if err != nil {
		return fmt.Errorf("error reading replies commentID: %s, error: %s", comment.CommentID, err)
	}

	defer resp.Body.Close()
	var replies ReplyResponse
	if err := json.NewDecoder(resp.Body).Decode(&replies); err != nil {
		return fmt.Errorf("error parsing json commentID: %s, error: %s", comment.CommentID, err)
	}
	if !replies.Success {
		data, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("status code is bad for replies commentID: %s, result: %s", comment.CommentID, string(data))
	}
	if *verbose {
		fmt.Printf("[COMMENT-CACHE] Successfully scanned commentID: %s from: %s replies: %d\n", comment.CommentID, url, len(replies.Val))
	}
	comment.Replies = replies.Val
	return nil
}

func decodeResponse[T []RawComment | []RawDiscussionList | RawDiscussion](resp *http.Response, def T) (T, error) {
	defer resp.Body.Close()
	res, err := io.ReadAll(resp.Body)
	if err != nil {
		return def, fmt.Errorf("error reading response, error: %s", err)
	}

	var response Response[T]
	if err := json.Unmarshal(res, &response); err != nil {
		return def, fmt.Errorf("error decoding json, error: %s", err)
	}
	if !response.Success {
		return def, fmt.Errorf("status code is bad, result: %s", res)
	}
	return response.Val, nil
}

// Returns number processed
func decodeComments(comments []RawComment, discussionID uint32, mangaName string) []Comment {
	commentArr := make([]Comment, len(comments))

	for _, comment := range comments {
		commentTime, err := time.Parse("2006-01-02 15:04:05", comment.TimeCommented)
		if err != nil {
			fmt.Println("[COMMENT-CACHE] Error parsing time:", err)
			numErrors.Add(1)
			continue
		}
		newcomment := Comment{
			ID:           conv[uint32](comment.CommentID),
			UserID:       conv[uint32](comment.UserID),
			Content:      comment.CommentContent,
			Likes:        conv[int16](comment.LikeCount),
			Timestamp:    commentTime,
			DiscussionID: discussionID,
			MangaName:    mangaName,
		}
		for _, reply := range comment.Replies {
			commentTime, err := time.Parse("2006-01-02 15:04:05", reply.TimeCommented)
			if err != nil {
				fmt.Println("[COMMENT-CACHE] Error parsing time:", err)
				numErrors.Add(1)
				continue
			}

			newreply := Reply{
				ID:        conv[uint32](reply.CommentID),
				UserID:    conv[uint32](reply.UserID),
				Content:   reply.CommentContent,
				Timestamp: commentTime,
				CommentID: conv[uint32](comment.CommentID),
			}
			newcomment.Replies = append(newcomment.Replies, newreply)
		}
		commentArr = append(commentArr, newcomment)
	}
	return commentArr
}