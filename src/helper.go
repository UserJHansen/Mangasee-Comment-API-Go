package main

import "strconv"

func conv[T uint | int | uint8 | uint16 | uint32 | uint64 | int8 | int16 | int32 | int64](str string) T {
	val, _ := strconv.Atoi(str)
	return T(val)
}

func cleanComments() {
	// Deduplicate the comments
	keys := make(map[int]bool)
	totalReplies := uint32(0)
	dedupedComments := []Comment{}
	for _, comment := range comments {
		if _, value := keys[int(comment.ID)]; !value {
			keys[int(comment.ID)] = true
			dedupedComments = append(dedupedComments, comment)
			totalReplies += uint32(len(comment.Replies))
		}
	}
	comments = dedupedComments
	commentNo.Add(float64(len(comments)) - commentCounterVal)
	commentCounterVal = float64(len(comments))

	replyNo.Add(float64(totalReplies) - replyCounterVal)
	replyCounterVal = float64(totalReplies)
}
