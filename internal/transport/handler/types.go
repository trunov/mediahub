package handler

type UploadImageParams struct {
	ItemID      int64  `validate:"required"`          // images.item_id (NOT NULL)
	SKU         string `validate:"omitempty,max=64"`  // images.sku
	Context     string `validate:"required,max=64"`   // images.context (NOT NULL)
	Description string `validate:"omitempty,max=255"` // images.description
	Project     string `validate:"required,max=64"`   // images.project (NOT NULL)
	OrderIndex  int64  `validate:"gte=0,lte=32767"`   // images.order_index (NOT NULL)

	// Options
	PreserveFilename bool // from query ?preserveFilename=1

	// Auth
	UserID int64 `validate:"required"`
}
