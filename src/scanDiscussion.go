package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

type RawDiscussion struct {
	Notification string
	PostContent  string
	PostID       string
	PostTitle    string
	PostType     string
	TimePosted   string
	UserID       string
	Username     string
	Comments     []RawComment
}
type RawDiscussionList struct {
	PostID       string
	CountComment string
	Username     string
	PostTitle    string
	PostType     string
	TimePosted   string
	CommentOrder string
	TimeOrder    bool
}

type DiscussionResponse Response[[]RawDiscussion]
type DiscussionListResponse Response[[]RawDiscussionList]

func scanDiscussion(id uint32) (RawDiscussion, error) {
	values := map[string]string{"id": fmt.Sprintf("%d", id)}
	json_data, _ := json.Marshal(values)

	resp, err := http.Post(*server+"discussion/post.get.php", "application/json",
		bytes.NewBuffer(json_data))

	if err != nil {
		fmt.Printf("[COMMENT-CACHE] Error reading comments for: %d, error: %s\n", id, err)
		return RawDiscussion{}, err
	}

	res, err := decodeResponse(resp, RawDiscussion{})

	if err != nil {
		fmt.Printf("[COMMENT-CACHE] Error reading comments for: %d, error: %s\n", id, err)
		return RawDiscussion{}, err
	}

	return res, nil
}

func scanAllDiscussions() error {
	start := time.Now().UnixMicro()
	fmt.Println("[COMMENT-CACHE] Starting discussion scan...")
	resp, err := http.Get(*server + "discussion/index.get.php")
	if err != nil {
		fmt.Println("[COMMENT-CACHE] Error getting Discussions:", err)
		return err
	}
	defer resp.Body.Close()

	rawDiscussions, err := decodeResponse[[]RawDiscussionList](resp, nil)
	if err != nil {
		fmt.Println("[COMMENT-CACHE] Error decoding Discussion list:", err)
		return err
	}
	fmt.Printf("[COMMENT-CACHE] Took %ds to get discussion list\n", (time.Now().UnixMicro()-start)/time.Second.Microseconds())

	// Get and deduplicate discussion ids
	ids := make(map[uint32]bool)
	dedupedIds := make([]uint32, len(discussionIds))
	copy(dedupedIds, discussionIds)
	for _, id := range discussionIds {
		ids[id] = true
	}
	for _, discussion := range rawDiscussions {
		id := conv[uint32](discussion.PostID)

		if !ids[id] {
			ids[id] = true
			dedupedIds = append(dedupedIds, id)
		}
	}
	discussionIds = dedupedIds
	start = time.Now().UnixMicro()

	// Thread limiting
	var wg sync.WaitGroup
	guard := make(chan struct{}, *procs)
	wg.Add(len(discussionIds))

	// Data storage
	newDiscussions := make([]Discussion, len(discussionIds))
	discussionComments := make([][]RawComment, len(discussionIds))
	newMap := []Username{}

	for i, discussion := range discussionIds {
		guard <- struct{}{}
		go func(id uint32, i int) {
			result, err := scanDiscussion(id)
			if err == nil {
				discussionComments[i] = result.Comments
				discussionTime, err := time.Parse("2006-01-02 15:04:05", result.TimePosted)
				if err != nil {
					fmt.Println("[COMMENT-CACHE] Error parsing time:", err)
					numErrors.Add(1)
				} else {
					newDiscussions[i] = Discussion{
						ID:        conv[uint32](result.PostID),
						UserID:    conv[uint32](result.UserID),
						Title:     result.PostTitle,
						Type:      toPostType(result.PostType),
						Content:   result.PostContent,
						Timestamp: discussionTime,
					}
					newMap = append(newMap, Username{ID: conv[uint32](result.UserID), Name: result.Username})

					if *verbose {
						fmt.Printf("[COMMENT-CACHE] (%d/%d) Successfully scanned %s\n", i, len(rawDiscussions), result.PostTitle)
					}
				}
			} else {
				numErrors.Add(1)
			}
			<-guard
			wg.Done()
		}(discussion, i)
	}
	wg.Wait()

	fmt.Printf("[COMMENT-CACHE] Took %ds to scan %d discussions\n", (time.Now().UnixMicro()-start)/time.Second.Microseconds(), len(discussionIds))
	fmt.Printf("[COMMENT-CACHE] That's an average of %dμs per discussion\n", (time.Now().UnixMicro()-start)/int64(len(rawDiscussions)))

	scanTime.WithLabelValues("discussion").Set(float64((time.Now().UnixMicro() - start) / time.Millisecond.Microseconds()))

	start = time.Now().UnixMicro()
	numberRequests := 0
	// Get replies for comments that exceed the limit
	for i, discussion := range discussionComments {
		for _, comment := range discussion {
			if int16(len(comment.Replies)) > comment.ReplyLimit {
				wg.Add(1)
				guard <- struct{}{}
				go func(name string, comment *RawComment) {
					err = getReplies(comment, "discussion/post.reply.get.php")
					if err != nil {
						numErrors.Add(1)
						fmt.Printf("[COMMENT-CACHE] On %s err: %s\n", name, err)
					}
					numberRequests++
					<-guard
					wg.Done()
				}(newDiscussions[i].Title, &comment)
			}
		}
	}
	wg.Wait()

	fmt.Printf("[COMMENT-CACHE] Took %dms to get replies for %d comments\n", (time.Now().UnixMicro()-start)/time.Millisecond.Microseconds(), numberRequests)
	fmt.Printf("[COMMENT-CACHE] That's an average of %dμs per reply\n", (time.Now().UnixMicro()-start)/int64(numberRequests))

	// create a rough map of UserIDs to usernames
	start = time.Now().UnixMicro()
	for _, discussion := range discussionComments {
		for _, comment := range discussion {
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

	fmt.Printf("[COMMENT-CACHE] Took %dμs to extract usernames\n", time.Now().UnixMicro()-start)

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

	fmt.Printf("[COMMENT-CACHE] Took %dμs to deduplicate %d users\n", time.Now().UnixMicro()-start, len(userMap))

	start = time.Now().UnixMicro()
	// Create a proper tree of comments and replies
	for i, discussion := range discussionComments {
		commentArr := decodeComments(discussion, uint32(discussionIds[i]), "")
		comments = append(comments, commentArr...)
	}

	cleanComments()

	fmt.Printf("[COMMENT-CACHE] Took %dμs to create a proper tree of comments and replies\n", time.Now().UnixMicro()-start)

	start = time.Now().UnixMicro()

	newDiscussions = append(discussions, newDiscussions...)
	keys = make(map[int]bool)
	dedupedDiscussions := []Discussion{}
	for _, entry := range newDiscussions {
		if _, value := keys[int(entry.ID)]; !value {
			keys[int(entry.ID)] = true
			dedupedDiscussions = append(dedupedDiscussions, entry)
		}
	}
	discussions = dedupedDiscussions

	fmt.Printf("[COMMENT-CACHE] Took %dμs to deduplicate %d discussions\n", time.Now().UnixMicro()-start, len(discussions))

	// Write to file
	return save()

}
