package main

import (
	"encoding/json"
	"fmt"
	"os"
)

// Save to the cache file
func save() error {
	json, err := json.MarshalIndent(&SaveFile{
		Comments:      comments,
		Users:         userMap,
		Discussions:   discussions,
		DiscussionIds: discussionIds,
	}, "", "  ")
	if err != nil {
		numErrors.Add(1)
		return err
	}

	fmt.Println("[COMMENT-CACHE] Saving to file")
	return os.WriteFile(*saveLoc, json, 0644)
}

// Read from the cache file
func load() error {
	text, err := os.ReadFile(*saveLoc)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}

		numErrors.Add(1)
		return err
	}
	var saveData SaveFile
	readerr := json.Unmarshal(text, &saveData)
	if readerr != nil {
		return readerr
	}
	comments = saveData.Comments
	userMap = saveData.Users
	discussions = saveData.Discussions
	discussionIds = saveData.DiscussionIds

	fmt.Println("[COMMENT-CACHE] Reading Config")
	return nil
}
