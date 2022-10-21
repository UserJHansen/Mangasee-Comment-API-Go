package main

type SaveFile struct {
	Comments []Comment  `json:"comments"`
	Users    []Username `json:"users"`
}
type Result[T any] struct {
	Status string `json:"status"`
	Result T      `json:"result"`
}

type Comment struct {
	ID           uint32  `json:"id"`
	UserID       uint32  `json:"user_id"`
	Content      string  `json:"content"`
	Likes        int16   `json:"likes"`
	Timestamp    uint64  `json:"timestamp"`
	DiscussionID uint32  `json:"discussion_id"`
	MangaName    string  `json:"manga_name"`
	Replies      []Reply `json:"replies"`
}
type Reply struct {
	ID        uint32 `json:"id"`
	UserID    uint32 `json:"user_id"`
	Content   string `json:"content"`
	Timestamp uint64 `json:"timestamp"`
	CommentID uint32 `json:"comment_id"`
}
type Username struct {
	ID   uint32 `json:"id"`
	Name string `json:"name"`
}
