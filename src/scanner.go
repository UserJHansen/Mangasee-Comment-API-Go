package main

import (
	"fmt"
	"time"
)

func spawnScanner() {
	go func() {
		for {
			if err := scanAllDiscussions(); err != nil {
				fmt.Println("[COMMENT-CACHE] failed to scan for discussions:", err)

				numErrors.Add(1)
			}
			if err := scanAllManga(); err != nil {
				fmt.Println("[COMMENT-CACHE] failed to scan for comments:", err)

				numErrors.Add(1)
			}
			time.Sleep(time.Duration(*interval * int(time.Minute.Nanoseconds())))
		}
	}()
}
