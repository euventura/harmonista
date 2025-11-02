package models

import "time"

type User struct {
	ID                     int    `gorm:"primary_key;autoIncrement" json:"id"`
	PasswordHash           string `gorm:"not null" json:"-"` // json:"-" prevents password from being exposed in API
	Email                  string `gorm:"unique;not null" json:"email"`
	EmailVerified          bool   `gorm:"default:false" json:"email_verified"`
	EmailVerificationToken string `json:"-"` // token for email verification
	SessionToken           string `json:"-"` // for session management
}

type Blog struct {
	ID           int    `gorm:"primary_key;autoIncrement" json:"id"`
	UserID       int    `gorm:"not null;index" json:"user_id"` // auto-filled
	Title        string `gorm:"not null" json:"title"`         //mandatory
	Description  string `gorm:"type:text" json:"description"`  // Description for homepage and SEO
	Subdomain    string `gorm:"unique;not null;index" json:"subdomain"`
	Nav          string `gorm:"type:text" json:"nav"`                      // Navigation links in markdown format
	Theme        string `gorm:"type:text" json:"theme"`                    // Optional - large CSS text
	IsListReader bool   `gorm:"default:false;index" json:"is_list_reader"` // always false until the user opts in
	IsAdult      bool   `gorm:"default:false" json:"is_adult"`             // always false until the user opts in
}

type Post struct {
	ID        uint       `gorm:"primary_key"`
	BlogID    int        `gorm:"not null;index" json:"blog_id"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `sql:"index" json:"deleted_at,omitempty"`
	Title     string     `gorm:"not null" json:"title"`
	Slug      string     `gorm:"not null;index" json:"slug"`
	Content   string     `gorm:"type:text" json:"content"`
	Draft     bool       `json:"draft"`
}

type Page struct {
	ID        uint       `gorm:"primary_key"`
	BlogID    int        `gorm:"not null;index" json:"blog_id"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `sql:"index" json:"deleted_at,omitempty"`
	Title     string     `gorm:"not null" json:"title"`
	Slug      string     `gorm:"not null;index" json:"slug"`
	Content   string     `gorm:"type:text" json:"content"`
	Draft     bool       `json:"draft"`
}

type Tag struct {
	ID    uint   `gorm:"primary_key"`
	Title string `gorm:"not null;index json:"title"`
}

type PostTag struct {
	ID     uint `gorm:"primary_key"`
	PostID int  `gorm:"not null;index" json:"post_id"`
	TagID  int  `gorm:"not null;index" json:"tag_id"`
}
