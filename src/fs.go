package main

import (
	"encoding/json"
	"fmt"
	"os"
)

// Save to the cache file
func save() error {
	json, err := json.MarshalIndent(&SaveFile{
		Comments: comments,
		Users:    userMap,
	}, "", "  ")
	if err != nil {
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

		return err
	}
	var saveData SaveFile
	readerr := json.Unmarshal(text, &saveData)
	if readerr != nil {
		return readerr
	}
	comments = saveData.Comments
	userMap = saveData.Users

	fmt.Println("[COMMENT-CACHE] Reading Config")
	return nil
}
