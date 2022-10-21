package main

import (
	"fmt"
	"time"
)

func spawnScanner() {
	go func() {
		if err := scanAllManga(); err != nil {
			fmt.Println("[COMMENT-CACHE] failed to scan for comments:", err)
		}
		time.Sleep(time.Duration(*interval * int(time.Minute.Nanoseconds())))
	}()
}
