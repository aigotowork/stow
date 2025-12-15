package main

import (
	"fmt"
	"log"
	"time"

	"github.com/aigotowork/stow"
)

// UserProfile demonstrates various struct tag usages
type UserProfile struct {
	// Basic field with JSON name mapping
	UserID string `json:"user_id"`

	// Field with different JSON name
	FullName string `json:"full_name"`

	// Omit empty values
	Nickname string `json:"nickname,omitempty"`

	// Large field stored as blob
	ProfilePicture []byte `json:"profile_picture" stow:"file"`

	// Force inline storage despite size
	SmallAvatar []byte `json:"small_avatar" stow:"inline"`

	// Nested struct
	Address Address `json:"address"`

	// Pointer to nested struct
	Preferences *Preferences `json:"preferences,omitempty"`

	// Array of nested structs
	SocialLinks []SocialLink `json:"social_links"`

	// Slice of basic types
	Tags []string `json:"tags"`

	// Map
	Metadata map[string]interface{} `json:"metadata"`

	// Time field
	JoinedAt time.Time `json:"joined_at"`

	// Ignored field (never serialized)
	temporaryData string `json:"-"`
}

// Address represents a user's address
type Address struct {
	Street  string `json:"street"`
	City    string `json:"city"`
	State   string `json:"state"`
	ZipCode string `json:"zip_code"`
	Country string `json:"country"`
}

// Preferences represents user preferences
type Preferences struct {
	Theme            string `json:"theme"`
	Language         string `json:"language"`
	EmailNotify      bool   `json:"email_notify"`
	PushNotify       bool   `json:"push_notify"`
	AutoSave         bool   `json:"auto_save"`
	ItemsPerPage     int    `json:"items_per_page"`
	DefaultView      string `json:"default_view"`
	PrivacyLevel     string `json:"privacy_level"`
	TwoFactorEnabled bool   `json:"two_factor_enabled"`
}

// SocialLink represents a social media link
type SocialLink struct {
	Platform string `json:"platform"`
	URL      string `json:"url"`
	Username string `json:"username"`
	Verified bool   `json:"verified"`
}

// Product demonstrates different field types
type Product struct {
	// String fields
	SKU         string `json:"sku"`
	Name        string `json:"name"`
	Description string `json:"description"`

	// Numeric fields
	Price    float64 `json:"price"`
	Quantity int     `json:"quantity"`
	Rating   float32 `json:"rating"`

	// Boolean fields
	InStock     bool `json:"in_stock"`
	Featured    bool `json:"featured"`
	OnSale      bool `json:"on_sale"`
	Downloadable bool `json:"downloadable"`

	// Array and slices
	Images      []string `json:"images"`
	Categories  []string `json:"categories"`
	Tags        []string `json:"tags"`

	// Large binary data
	Manual    []byte `json:"manual" stow:"file"`      // PDF manual stored as blob
	Thumbnail []byte `json:"thumbnail" stow:"inline"` // Small thumbnail inlined

	// Nested struct
	Dimensions ProductDimensions `json:"dimensions"`

	// Pointer fields
	Discount *Discount `json:"discount,omitempty"`

	// Time fields
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
	LaunchedAt time.Time `json:"launched_at"`

	// Map for custom attributes
	CustomFields map[string]interface{} `json:"custom_fields"`
}

// ProductDimensions represents product size
type ProductDimensions struct {
	Length float64 `json:"length"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
	Weight float64 `json:"weight"`
	Unit   string  `json:"unit"`
}

// Discount represents a product discount
type Discount struct {
	Percentage float64   `json:"percentage"`
	StartDate  time.Time `json:"start_date"`
	EndDate    time.Time `json:"end_date"`
	Code       string    `json:"code"`
}

func main() {
	storePath := "./data/struct_tags_example"

	store, err := stow.Open(storePath)
	if err != nil {
		log.Fatal(err)
	}
	defer store.Close()

	fmt.Println("=== Struct Tags Examples ===")
	fmt.Println()

	// Example 1: User profiles with various field types
	demonstrateUserProfiles(store)

	// Example 2: Products with complex nested structures
	demonstrateProducts(store)

	// Example 3: Force file vs force inline
	demonstrateStorageOptions(store)

	fmt.Println("\n=== All struct tag examples completed! ===")
	fmt.Printf("\nData stored at: %s\n", storePath)
	fmt.Println("\nKey concepts demonstrated:")
	fmt.Println("  - JSON field naming with 'json' tags")
	fmt.Println("  - Omit empty with 'omitempty'")
	fmt.Println("  - Force blob storage with 'stow:\"file\"'")
	fmt.Println("  - Force inline storage with 'stow:\"inline\"'")
	fmt.Println("  - Nested structs and pointers")
	fmt.Println("  - Arrays, slices, and maps")
	fmt.Println("  - Time fields")}

func demonstrateUserProfiles(store stow.Store) {
	fmt.Println("1. User Profiles (Various Field Types)")

	ns, err := store.GetNamespace("users")
	if err != nil {
		log.Fatal(err)
	}

	// Create sample profile picture (simulated)
	profilePic := make([]byte, 50000) // 50KB image
	for i := range profilePic {
		profilePic[i] = byte(i % 256)
	}

	// Create small avatar (simulated)
	smallAvatar := make([]byte, 1500) // 1.5KB thumbnail
	for i := range smallAvatar {
		smallAvatar[i] = byte((i * 3) % 256)
	}

	profile := UserProfile{
		UserID:         "user-12345",
		FullName:       "Jane Doe",
		Nickname:       "jdoe",
		ProfilePicture: profilePic,
		SmallAvatar:    smallAvatar,
		Address: Address{
			Street:  "123 Main Street",
			City:    "San Francisco",
			State:   "CA",
			ZipCode: "94102",
			Country: "USA",
		},
		Preferences: &Preferences{
			Theme:            "dark",
			Language:         "en-US",
			EmailNotify:      true,
			PushNotify:       false,
			AutoSave:         true,
			ItemsPerPage:     25,
			DefaultView:      "grid",
			PrivacyLevel:     "friends",
			TwoFactorEnabled: true,
		},
		SocialLinks: []SocialLink{
			{
				Platform: "Twitter",
				URL:      "https://twitter.com/jdoe",
				Username: "@jdoe",
				Verified: true,
			},
			{
				Platform: "GitHub",
				URL:      "https://github.com/jdoe",
				Username: "jdoe",
				Verified: false,
			},
			{
				Platform: "LinkedIn",
				URL:      "https://linkedin.com/in/jdoe",
				Username: "jane-doe",
				Verified: true,
			},
		},
		Tags: []string{"developer", "open-source", "golang"},
		Metadata: map[string]interface{}{
			"account_type":    "premium",
			"last_login":      "2024-12-14",
			"login_count":     156,
			"email_confirmed": true,
		},
		JoinedAt:      time.Now().Add(-365 * 24 * time.Hour), // 1 year ago
		temporaryData: "This will not be stored",
	}

	if err := ns.Put(profile.UserID, profile); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("   ✓ Stored user profile: %s\n", profile.FullName)
	fmt.Printf("     - Large profile pic: %.1f KB (stored as blob)\n", float64(len(profile.ProfilePicture))/1024)
	fmt.Printf("     - Small avatar: %.1f KB (stored inline)\n", float64(len(profile.SmallAvatar))/1024)
	fmt.Printf("     - Address: %s, %s\n", profile.Address.City, profile.Address.Country)
	fmt.Printf("     - Social links: %d\n", len(profile.SocialLinks))
	fmt.Printf("     - Tags: %v\n", profile.Tags)

	// Retrieve and verify
	var retrieved UserProfile
	if err := ns.Get(profile.UserID, &retrieved); err != nil {
		log.Fatal(err)
	}

	fmt.Println("   ✓ Retrieved profile successfully")
	fmt.Printf("     - Preferences theme: %s\n", retrieved.Preferences.Theme)
	fmt.Printf("     - Profile picture size: %d bytes\n", len(retrieved.ProfilePicture))
	fmt.Printf("     - Metadata entries: %d\n", len(retrieved.Metadata))
	fmt.Println()
}

func demonstrateProducts(store stow.Store) {
	fmt.Println("2. Products (Complex Nested Structures)")

	ns, err := store.GetNamespace("products")
	if err != nil {
		log.Fatal(err)
	}

	// Simulate PDF manual
	manual := make([]byte, 200000) // 200KB PDF
	for i := range manual {
		manual[i] = byte(i % 256)
	}

	// Simulate thumbnail
	thumbnail := make([]byte, 800) // 800 bytes
	for i := range thumbnail {
		thumbnail[i] = byte((i * 5) % 256)
	}

	product := Product{
		SKU:         "PROD-2024-001",
		Name:        "Wireless Headphones Pro",
		Description: "Premium wireless headphones with active noise cancellation",
		Price:       299.99,
		Quantity:    150,
		Rating:      4.7,
		InStock:     true,
		Featured:    true,
		OnSale:      true,
		Downloadable: false,
		Images: []string{
			"image1.jpg",
			"image2.jpg",
			"image3.jpg",
		},
		Categories: []string{"Electronics", "Audio", "Headphones"},
		Tags:       []string{"wireless", "noise-cancelling", "premium", "bluetooth"},
		Manual:     manual,
		Thumbnail:  thumbnail,
		Dimensions: ProductDimensions{
			Length: 20.5,
			Width:  18.0,
			Height: 8.5,
			Weight: 0.25,
			Unit:   "cm/kg",
		},
		Discount: &Discount{
			Percentage: 15.0,
			StartDate:  time.Now().Add(-7 * 24 * time.Hour),
			EndDate:    time.Now().Add(7 * 24 * time.Hour),
			Code:       "WINTER15",
		},
		CreatedAt:  time.Now().Add(-90 * 24 * time.Hour),
		UpdatedAt:  time.Now().Add(-2 * 24 * time.Hour),
		LaunchedAt: time.Now().Add(-60 * 24 * time.Hour),
		CustomFields: map[string]interface{}{
			"battery_life":     "30 hours",
			"charging_time":    "2 hours",
			"bluetooth_version": 5.2,
			"warranty_years":   2,
			"color_options":    []string{"Black", "Silver", "Gold"},
		},
	}

	if err := ns.Put(product.SKU, product); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("   ✓ Stored product: %s\n", product.Name)
	fmt.Printf("     - SKU: %s\n", product.SKU)
	fmt.Printf("     - Price: $%.2f (%.0f%% off)\n", product.Price, product.Discount.Percentage)
	fmt.Printf("     - Rating: %.1f/5.0\n", product.Rating)
	fmt.Printf("     - Stock: %d units\n", product.Quantity)
	fmt.Printf("     - Images: %d\n", len(product.Images))
	fmt.Printf("     - Manual size: %.1f KB (stored as blob)\n", float64(len(product.Manual))/1024)
	fmt.Printf("     - Thumbnail: %d bytes (stored inline)\n", len(product.Thumbnail))
	fmt.Printf("     - Dimensions: %.1f x %.1f x %.1f %s\n",
		product.Dimensions.Length,
		product.Dimensions.Width,
		product.Dimensions.Height,
		product.Dimensions.Unit)

	// Retrieve and verify all nested structures
	var retrieved Product
	if err := ns.Get(product.SKU, &retrieved); err != nil {
		log.Fatal(err)
	}

	fmt.Println("   ✓ Retrieved product with all nested data:")
	fmt.Printf("     - Categories: %v\n", retrieved.Categories)
	fmt.Printf("     - Tags: %v\n", retrieved.Tags)
	fmt.Printf("     - Discount code: %s\n", retrieved.Discount.Code)
	fmt.Printf("     - Custom fields: %d entries\n", len(retrieved.CustomFields))
	fmt.Println()
}

func demonstrateStorageOptions(store stow.Store) {
	fmt.Println("3. Storage Options (Force File vs Force Inline)")

	ns, err := store.GetNamespace("options")
	if err != nil {
		log.Fatal(err)
	}

	// Medium-sized data (3KB)
	mediumData := make([]byte, 3000)
	for i := range mediumData {
		mediumData[i] = byte(i % 256)
	}

	type StorageExample struct {
		Name string `json:"name"`
		Data []byte `json:"data"`
	}

	// 1. Default behavior (size-based decision)
	example1 := StorageExample{
		Name: "default-behavior",
		Data: mediumData,
	}
	if err := ns.Put("example1", example1); err != nil {
		log.Fatal(err)
	}
	fmt.Println("   ✓ Example 1: Default behavior (3KB data)")
	fmt.Println("     → Decision based on BlobThreshold config")

	// 2. Force file storage (using tag)
	type ForceFileExample struct {
		Name string `json:"name"`
		Data []byte `json:"data" stow:"file"`
	}
	example2 := ForceFileExample{
		Name: "force-file",
		Data: mediumData,
	}
	if err := ns.Put("example2", example2); err != nil {
		log.Fatal(err)
	}
	fmt.Println("   ✓ Example 2: Force file with struct tag")
	fmt.Println("     → Stored as blob regardless of size")

	// 3. Force inline storage (using tag)
	type ForceInlineExample struct {
		Name string `json:"name"`
		Data []byte `json:"data" stow:"inline"`
	}
	example3 := ForceInlineExample{
		Name: "force-inline",
		Data: mediumData,
	}
	if err := ns.Put("example3", example3); err != nil {
		log.Fatal(err)
	}
	fmt.Println("   ✓ Example 3: Force inline with struct tag")
	fmt.Println("     → Stored inline regardless of size")

	// 4. Force file with PutOption
	example4 := StorageExample{
		Name: "force-file-option",
		Data: mediumData,
	}
	if err := ns.Put("example4", example4, stow.WithForceFile()); err != nil {
		log.Fatal(err)
	}
	fmt.Println("   ✓ Example 4: Force file with PutOption")
	fmt.Println("     → WithForceFile() option overrides default")

	// 5. Force inline with PutOption
	example5 := StorageExample{
		Name: "force-inline-option",
		Data: mediumData,
	}
	if err := ns.Put("example5", example5, stow.WithForceInline()); err != nil {
		log.Fatal(err)
	}
	fmt.Println("   ✓ Example 5: Force inline with PutOption")
	fmt.Println("     → WithForceInline() option overrides default")

	fmt.Println("\n   ℹ Priority order:")
	fmt.Println("     1. PutOption (WithForceFile/WithForceInline) - highest")
	fmt.Println("     2. Struct tag (stow:\"file\"/stow:\"inline\")")
	fmt.Println("     3. Type detection (io.Reader)")
	fmt.Println("     4. Size threshold (BlobThreshold config) - lowest")
	fmt.Println()
}
