package entities

import "time"

type Image struct {
	ID               int64     `json:"id"`
	UserID           int64     `json:"user_id"`
	ItemID           int64     `json:"item_id"`
	SKU              *string   `json:"sku,omitempty"`
	Context          string    `json:"context"`
	Description      *string   `json:"description,omitempty"`
	Width            int16     `json:"width"`
	Height           int16     `json:"height"`
	Project          string    `json:"project"`
	Size             int32     `json:"size"`
	Key              string    `json:"key"`
	WebPKey          *string   `json:"webp_key,omitempty"`
	MimeType         string    `json:"mime_type"`
	IsDeleted        bool      `json:"is_deleted"`
	OrderIndex       int16     `json:"order_index"`
	CreatedTimestamp time.Time `json:"created_timestamp"`
	UpdatedTimestamp time.Time `json:"updated_timestamp"`
}
