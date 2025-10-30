package queue

// ConvertJob is what we push to Redis Streams.
// No bytes here—workers fetch by ObjectKey.
type ConvertJob struct {
	ObjectKey   string `json:"object_key"`
	ContentType string `json:"content_type"`
	Ext         string `json:"ext"`                // ".jpg" | ".jpeg" | ".png"
	WebPKey     string `json:"webp_key,omitempty"` // optional override (defaults to ObjectKey + ".webp")
}
