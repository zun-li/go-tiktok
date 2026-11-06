package account

import "time"

type Account struct {
	ID               uint       `gorm:"primaryKey" json:"id"`
	Username         string     `gorm:"size:32;not null;uniqueIndex" json:"username"`
	Password         string     `gorm:"column:password_hash;size:255;not null" json:"-"`
	Token            string     `json:"-"`
	RefreshToken     string     `gorm:"size:64;index" json:"-"`
	RefreshExpiresAt *time.Time `json:"-"`
	CreatedAt        time.Time  `json:"created_at"`
	AvatarURL        string     `gorm:"type:varchar(512)" json:"avatar_url,omitempty"`
	Bio              string     `gorm:"type:varchar(255)" json:"bio,omitempty"`
}

func (Account) TableName() string { return "users" }

type CreateAccountRequest struct {
	Username string `json:"username" binding:"required,min=3,max=32,alphanum"`
	Password string `json:"password" binding:"required,min=8"`
}

type RenameRequest struct {
	NewUsername string `json:"new_username" binding:"required,min=3,max=32,alphanum"`
}

type FindByIDRequest struct {
	ID uint `json:"id"`
}

type FindByIDResponse struct {
	ID        uint   `json:"id"`
	Username  string `json:"username"`
	AvatarURL string `json:"avatar_url,omitempty"`
	Bio       string `json:"bio,omitempty"`
}

type FindByUsernameRequest struct {
	Username string `json:"username" binding:"required,min=3,max=32,alphanum"`
}

type FindByUsernameResponse struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
}

type ChangePasswordRequest struct {
	Username    string `json:"username"`
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=8"`
}

type LoginRequest struct {
	Username string `json:"username" binding:"required,min=3,max=32,alphanum"`
	Password string `json:"password" binding:"required,min=8"`
}

type LoginResponse struct {
	Token        string `json:"token"`
	RefreshToken string `json:"refresh_token"`
	AccountID    uint   `json:"account_id"`
	Username     string `json:"username"`
}

type UpdateProfileRequest struct {
	AvatarURL *string `json:"avatar_url"`
	Bio       *string `json:"bio"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type GetProfileRequest struct {
	AccountID uint `json:"account_id"`
}

type GetProfileResponse struct {
	Account       FindByIDResponse `json:"account"`
	VideoCount    int64            `json:"video_count"`
	TotalLikes    int64            `json:"total_likes"`
	FollowerCount int64            `json:"follower_count"`
	VloggerCount  int64            `json:"vlogger_count"`
}
