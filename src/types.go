package main

import (
	"fmt"
	"strings"
	"time"
)

type SaveFile struct {
	Comments      []Comment    `json:"comments"`
	Users         []Username   `json:"users"`
	Discussions   []Discussion `json:"discussions"`
	DiscussionIds []uint32     `json:"discussion_ids"`
	Deleted       []uint32     `json:"deleted"`
}
var deletedErr = fmt.Errorf("deleted")

type Result[T any] struct {
	Status string `json:"status"`
	Result T      `json:"result"`
}

type PostType uint8

const (
	General PostType = iota
	Request
	Question
	Announcement
)

func (s PostType) String() string {
	types := [...]string{"general", "request", "question", "announcement"}
	if !s.IsValid() {
		return fmt.Sprintf("postType(%d)", int(s))
	}
	return types[s]
}
func toPostType(s string) PostType {
	switch strings.ToLower(s) {
	case "general":
		return General
	case "request":
		return Request
	case "question":
		return Question
	case "announcement":
		return Announcement
	default:
		return General
	}
}
func (s PostType) IsValid() bool {
	switch s {
	case General, Request, Question, Announcement:
		return true
	}
	return false
}

type Discussion struct {
	ID        uint32    `json:"id"`
	UserID    uint32    `json:"user_id"`
	Title     string    `json:"title"`
	Type      PostType  `json:"type"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
}
type Comment struct {
	ID           uint32    `json:"id"`
	UserID       uint32    `json:"user_id"`
	Content      string    `json:"content"`
	Likes        int16     `json:"likes"`
	Timestamp    time.Time `json:"timestamp"`
	DiscussionID uint32    `json:"discussion_id"`
	MangaName    string    `json:"manga_name"`
	Replies      []Reply   `json:"replies"`
}
type Reply struct {
	ID        uint32    `json:"id"`
	UserID    uint32    `json:"user_id"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
}
type Username struct {
	ID   uint32 `json:"id"`
	Name string `json:"name"`
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
