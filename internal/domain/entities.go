package domain

import "time"

type TenantStatus string

type UserRole string

type UserStatus string

const (
	RoleAdmin  UserRole = "admin"
	RoleEditor UserRole = "editor"
	RoleViewer UserRole = "viewer"
)

type Tenant struct {
	ID        int64
	Name      string
	Domain    string
	Status    string
	CreatedAt time.Time
}

type User struct {
	ID           int64
	TenantID     int64
	Email        string
	PasswordHash string
	Role         UserRole
	Status       string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type Link struct {
	ID         int64
	TenantID   int64
	Slug       string
	TargetURL  string
	IsActive   bool
	ExpireAt   *time.Time
	MaxClicks  *int64
	CreatedBy  int64
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type LinkMeta struct {
	LinkID      int64
	Title       string
	Description string
	OGImage     string
	NoPreview   bool
}

type Device string

const (
	DeviceIOS     Device = "ios"
	DeviceAndroid Device = "android"
	DeviceDesktop Device = "desktop"
)

type LinkRule struct {
	ID          int64
	LinkID      int64
	CountryCode *string
	Device      *Device
	Weight      *int
	TargetURL   string
}

type ClickAgg struct {
	Day      time.Time
	Slug     string
	TenantID int64
	Total    int64
}
