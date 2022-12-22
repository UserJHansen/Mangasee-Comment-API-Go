package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"
)

type SearchResponse struct {
	IndexName      string   `json:"i"`
	StringName     string   `json:"s"`
	AlternateNames []string `json:"a"`
}

func scanManga(manga SearchResponse) ([]RawComment, error) {
	values := map[string]string{"IndexName": manga.IndexName}
	json_data, _ := json.Marshal(values)

	resp, err := http.Post(*server+"manga/comment.get.php", "application/json",
		bytes.NewBuffer(json_data))

	if err != nil {
		Printf("[COMMENT-CACHE] Error reading comments for: %s, error: %s\n", manga.IndexName, err)
		return nil, err
	}

	res, err := decodeResponse[[]RawComment](resp, nil)

	if err != nil {
		Printf("[COMMENT-CACHE] Error reading comments for: %s, error: %s\n", manga.IndexName, err)
		return nil, err
	}

	return res, nil
}

func scanAllManga() error {
	start := time.Now().UnixMicro()
	Println("[COMMENT-CACHE] Starting manga scan...")
	resp, err := http.Get(*server + "_search.php")
	if err != nil {
		Println("[COMMENT-CACHE] Error getting Manga:", err)
		return err
	}
	defer resp.Body.Close()

	var mangas []SearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&mangas); err != nil {
		Println("[COMMENT-CACHE] Error decoding Manga:", err)
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
					Printf("[COMMENT-CACHE] (%d/%d) Successfully scanned %s\n", i, len(mangas), manga.IndexName)
				}
				commentResults[i] = result
			} else {
				numErrors.Add(1)
			}
			<-guard
			wg.Done()
		}(manga, i)
	}
	wg.Wait()

	if *timing {
		Printf("[COMMENT-CACHE] Took %ds to scan %d mangas\n", (time.Now().UnixMicro()-start)/time.Second.Microseconds(), len(mangas))
		Printf("[COMMENT-CACHE] That's an average of %dμs per manga\n", (time.Now().UnixMicro()-start)/int64(len(mangas)))
	}

	scanTime.WithLabelValues("manga").Set(float64((time.Now().UnixMicro() - start) / time.Millisecond.Microseconds()))
	numManga.Set(float64(len(mangas)))

	start = time.Now().UnixMicro()
	numberRequests := 0
	// Get replies for comments that exceed the limit
	for i, manga := range commentResults {
		for _, comment := range manga {
			if int16(len(comment.Replies)) > comment.ReplyLimit {
				wg.Add(1)
				numberRequests++
				guard <- struct{}{}
				go func(name string, comment RawComment) {
					err := getReplies(comment, "manga/reply.get.php")
					if err != nil {
						numErrors.Add(1)
						Printf("[COMMENT-CACHE] On %s, err: %s\n", name, err)
					}
					<-guard
					wg.Done()
				}(mangas[i].IndexName, comment)
			}
		}
	}
	wg.Wait()

	if *timing {
		Printf("[COMMENT-CACHE] Took %ds to get replies for %d comments\n", (time.Now().UnixMicro()-start)/time.Second.Microseconds(), numberRequests)
		Printf("[COMMENT-CACHE] That's an average of %dμs per reply\n", (time.Now().UnixMicro()-start)/int64(numberRequests))
	}

	// create a rough map of UserIDs to usernames
	start = time.Now().UnixMicro()
	newMap := []Username{}
	for _, manga := range commentResults {
		for _, comment := range manga {
			for _, reply := range comment.Replies {
				newMap = append(newMap, Username{
					ID:   unsafeConv[uint32](reply.UserID),
					Name: reply.Username,
				})
			}
			newMap = append(newMap, Username{
				ID:   unsafeConv[uint32](comment.UserID),
				Name: comment.Username,
			})
		}
	}

	if *timing {
		Printf("[COMMENT-CACHE] Took %dms to extract usernames\n", (time.Now().UnixMicro()-start)/time.Millisecond.Microseconds())
	}

	start = time.Now().UnixMicro()
	// Deduplicate the map and update Prom
	newMap = append(newMap, userMap...)
	keys := make(map[int]bool)
	max := uint32(0)
	dedupedUsers := []Username{}
	for _, entry := range newMap {
		if _, value := keys[int(entry.ID)]; !value {
			keys[int(entry.ID)] = true
			dedupedUsers = append(dedupedUsers, entry)
			if entry.ID > max {
				max = entry.ID
			}
		}
	}
	userMap = dedupedUsers
	userNo.Add(float64(len(userMap)) - userCounterVal)
	userCounterVal = float64(len(userMap))
	totalPossibleUsers.Add(float64(max) - possibleUserVal)
	possibleUserVal = float64(max)

	if *timing {
		Printf("[COMMENT-CACHE] Took %dμs to deduplicate %d users\n", time.Now().UnixMicro()-start, len(userMap))
	}

	start = time.Now().UnixMicro()
	// Create a proper tree of comments and replies
	for i, manga := range commentResults {
		commentArr := decodeComments(manga, 0, strings.ToLower(mangas[i].IndexName))
		comments = append(comments, commentArr...)
	}

	cleanComments()

	if *timing {
		Printf("[COMMENT-CACHE] Took %dms to create a proper tree of comments and replies\n", (time.Now().UnixMicro()-start)/time.Millisecond.Microseconds())
	}

	// Write to file
	return save()
}
