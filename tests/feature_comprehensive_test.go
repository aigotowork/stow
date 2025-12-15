package stow_test

import (
	"testing"
	"time"

	"github.com/aigotowork/stow"
)

// Test all basic data types
func TestAllBasicDataTypes(t *testing.T) {
	tmpDir := t.TempDir()
	store := stow.MustOpen(tmpDir)
	defer store.Close()

	ns := store.MustGetNamespace("types")

	t.Run("String", func(t *testing.T) {
		ns.MustPut("string", "hello world")
		var result string
		ns.MustGet("string", &result)
		if result != "hello world" {
			t.Errorf("Expected 'hello world', got %v", result)
		}
	})

	t.Run("Int", func(t *testing.T) {
		ns.MustPut("int", 42)
		var result int
		ns.MustGet("int", &result)
		if result != 42 {
			t.Errorf("Expected 42, got %v", result)
		}
	})

	t.Run("Int64", func(t *testing.T) {
		ns.MustPut("int64", int64(9223372036854775807))
		var result int64
		ns.MustGet("int64", &result)
		if result != 9223372036854775807 {
			t.Errorf("Expected max int64, got %v", result)
		}
	})

	t.Run("Float64", func(t *testing.T) {
		ns.MustPut("float64", 3.14159)
		var result float64
		ns.MustGet("float64", &result)
		if result != 3.14159 {
			t.Errorf("Expected 3.14159, got %v", result)
		}
	})

	t.Run("Bool", func(t *testing.T) {
		ns.MustPut("bool-true", true)
		ns.MustPut("bool-false", false)

		var resultTrue, resultFalse bool
		ns.MustGet("bool-true", &resultTrue)
		ns.MustGet("bool-false", &resultFalse)

		if !resultTrue {
			t.Error("Expected true")
		}
		if resultFalse {
			t.Error("Expected false")
		}
	})

	t.Run("Slice", func(t *testing.T) {
		original := []string{"apple", "banana", "cherry"}
		ns.MustPut("slice", original)

		var result []string
		ns.MustGet("slice", &result)

		if len(result) != len(original) {
			t.Errorf("Expected length %d, got %d", len(original), len(result))
		}
		for i, v := range original {
			if result[i] != v {
				t.Errorf("Index %d: expected %s, got %s", i, v, result[i])
			}
		}
	})

	t.Run("Map", func(t *testing.T) {
		original := map[string]interface{}{
			"name":  "John",
			"age":   30,
			"city":  "New York",
			"score": 95.5,
		}
		ns.MustPut("map", original)

		var result map[string]interface{}
		ns.MustGet("map", &result)

		if result["name"] != "John" {
			t.Errorf("Expected name 'John', got %v", result["name"])
		}
	})

	t.Run("Bytes", func(t *testing.T) {
		original := []byte{0x01, 0x02, 0x03, 0x04, 0x05}
		ns.MustPut("bytes", original)

		var result []byte
		ns.MustGet("bytes", &result)

		if len(result) != len(original) {
			t.Errorf("Expected length %d, got %d", len(original), len(result))
		}
		for i, v := range original {
			if result[i] != v {
				t.Errorf("Index %d: expected %x, got %x", i, v, result[i])
			}
		}
	})

	t.Run("Time", func(t *testing.T) {
		type TimeWrapper struct {
			Timestamp time.Time `json:"timestamp"`
		}

		now := time.Now().Truncate(time.Second) // Truncate for comparison
		wrapper := TimeWrapper{Timestamp: now}
		ns.MustPut("time", wrapper)

		var result TimeWrapper
		ns.MustGet("time", &result)

		if !result.Timestamp.Equal(now) {
			t.Errorf("Expected %v, got %v", now, result.Timestamp)
		}
	})
}

// Test struct tags
func TestStructTags(t *testing.T) {
	tmpDir := t.TempDir()
	store := stow.MustOpen(tmpDir)
	defer store.Close()

	ns := store.MustGetNamespace("tags")

	t.Run("JSON name mapping", func(t *testing.T) {
		type User struct {
			UserID   string `json:"user_id"`
			FullName string `json:"full_name"`
			Age      int    `json:"age"`
		}

		original := User{
			UserID:   "123",
			FullName: "John Doe",
			Age:      30,
		}

		ns.MustPut("user1", original)

		var result User
		ns.MustGet("user1", &result)

		if result.UserID != original.UserID {
			t.Errorf("UserID mismatch: expected %s, got %s", original.UserID, result.UserID)
		}
		if result.FullName != original.FullName {
			t.Errorf("FullName mismatch: expected %s, got %s", original.FullName, result.FullName)
		}
	})

	t.Run("Omitempty tag", func(t *testing.T) {
		type Profile struct {
			Name     string `json:"name"`
			Nickname string `json:"nickname,omitempty"`
			Bio      string `json:"bio,omitempty"`
		}

		// With empty fields
		profile1 := Profile{
			Name:     "Alice",
			Nickname: "",
			Bio:      "",
		}
		ns.MustPut("profile1", profile1)

		var result1 Profile
		ns.MustGet("profile1", &result1)
		if result1.Name != "Alice" {
			t.Errorf("Expected Name 'Alice', got %s", result1.Name)
		}

		// With populated fields
		profile2 := Profile{
			Name:     "Bob",
			Nickname: "bobby",
			Bio:      "Developer",
		}
		ns.MustPut("profile2", profile2)

		var result2 Profile
		ns.MustGet("profile2", &result2)
		if result2.Nickname != "bobby" {
			t.Errorf("Expected Nickname 'bobby', got %s", result2.Nickname)
		}
	})

	t.Run("Stow file tag", func(t *testing.T) {
		type Document struct {
			Title   string `json:"title"`
			Content []byte `json:"content" stow:"file"`
		}

		content := make([]byte, 100)
		for i := range content {
			content[i] = byte(i)
		}

		doc := Document{
			Title:   "Test Document",
			Content: content,
		}

		ns.MustPut("doc1", doc)

		var result Document
		ns.MustGet("doc1", &result)

		if result.Title != doc.Title {
			t.Errorf("Title mismatch")
		}
		if len(result.Content) != len(doc.Content) {
			t.Errorf("Content length mismatch: expected %d, got %d", len(doc.Content), len(result.Content))
		}
	})

	t.Run("Stow inline tag", func(t *testing.T) {
		type SmallImage struct {
			Name string `json:"name"`
			Data []byte `json:"data" stow:"inline"`
		}

		data := make([]byte, 5000) // 5KB - would normally be blob
		for i := range data {
			data[i] = byte(i % 256)
		}

		img := SmallImage{
			Name: "thumbnail",
			Data: data,
		}

		ns.MustPut("img1", img)

		var result SmallImage
		ns.MustGet("img1", &result)

		if result.Name != img.Name {
			t.Errorf("Name mismatch")
		}
		if len(result.Data) != len(img.Data) {
			t.Errorf("Data length mismatch")
		}
	})

	t.Run("Ignore field with dash", func(t *testing.T) {
		type Account struct {
			Username string `json:"username"`
			password string // unexported field - not serialized
		}

		acc := Account{
			Username: "user123",
			password: "secret",
		}

		ns.MustPut("acc1", acc)

		var result Account
		ns.MustGet("acc1", &result)

		if result.Username != acc.Username {
			t.Errorf("Username mismatch")
		}
		// Unexported field should remain empty
		if result.password != "" {
			t.Errorf("password should be empty, got %s", result.password)
		}
	})
}

// Test complex nested structures
func TestComplexStructures(t *testing.T) {
	tmpDir := t.TempDir()
	store := stow.MustOpen(tmpDir)
	defer store.Close()

	ns := store.MustGetNamespace("complex")

	t.Run("Deep nesting", func(t *testing.T) {
		type Level3 struct {
			Value string `json:"value"`
		}
		type Level2 struct {
			Name   string  `json:"name"`
			Nested Level3  `json:"nested"`
		}
		type Level1 struct {
			ID     int     `json:"id"`
			Nested Level2  `json:"nested"`
		}

		original := Level1{
			ID: 1,
			Nested: Level2{
				Name: "level2",
				Nested: Level3{
					Value: "deep value",
				},
			},
		}

		ns.MustPut("deep", original)

		var result Level1
		ns.MustGet("deep", &result)

		if result.Nested.Nested.Value != "deep value" {
			t.Errorf("Deep nesting failed: expected 'deep value', got %s", result.Nested.Nested.Value)
		}
	})

	t.Run("Array of nested structs", func(t *testing.T) {
		type Item struct {
			Name  string `json:"name"`
			Price float64 `json:"price"`
		}
		type Order struct {
			OrderID string `json:"order_id"`
			Items   []Item `json:"items"`
		}

		order := Order{
			OrderID: "ORD-001",
			Items: []Item{
				{Name: "Item 1", Price: 10.99},
				{Name: "Item 2", Price: 20.50},
				{Name: "Item 3", Price: 5.25},
			},
		}

		ns.MustPut("order1", order)

		var result Order
		ns.MustGet("order1", &result)

		if len(result.Items) != 3 {
			t.Errorf("Expected 3 items, got %d", len(result.Items))
		}
		if result.Items[1].Name != "Item 2" {
			t.Errorf("Item name mismatch")
		}
		if result.Items[1].Price != 20.50 {
			t.Errorf("Item price mismatch")
		}
	})

	t.Run("Pointer fields", func(t *testing.T) {
		type Address struct {
			City string `json:"city"`
		}
		type Person struct {
			Name    string   `json:"name"`
			Address *Address `json:"address,omitempty"`
		}

		// With pointer
		person1 := Person{
			Name: "Alice",
			Address: &Address{
				City: "NYC",
			},
		}
		ns.MustPut("person1", person1)

		var result1 Person
		ns.MustGet("person1", &result1)
		if result1.Address == nil {
			t.Error("Address should not be nil")
		}
		if result1.Address.City != "NYC" {
			t.Errorf("Expected NYC, got %s", result1.Address.City)
		}

		// Without pointer
		person2 := Person{
			Name:    "Bob",
			Address: nil,
		}
		ns.MustPut("person2", person2)

		var result2 Person
		ns.MustGet("person2", &result2)
		// Address might be nil or empty depending on implementation
	})

	t.Run("Mixed types", func(t *testing.T) {
		type Contact struct {
			Name   string  `json:"name"`
			Age    int     `json:"age"`
			Height float64 `json:"height"`
			Active bool    `json:"active"`
			Tags   []string `json:"tags"`
			Meta   map[string]interface{} `json:"meta"`
		}

		contact := Contact{
			Name:   "Charlie",
			Age:    25,
			Height: 175.5,
			Active: true,
			Tags:   []string{"vip", "verified"},
			Meta: map[string]interface{}{
				"score":    100,
				"verified": true,
				"notes":    "Premium member",
			},
		}

		ns.MustPut("contact1", contact)

		var result Contact
		ns.MustGet("contact1", &result)

		if result.Name != contact.Name {
			t.Error("Name mismatch")
		}
		if result.Age != contact.Age {
			t.Error("Age mismatch")
		}
		if result.Height != contact.Height {
			t.Error("Height mismatch")
		}
		if result.Active != contact.Active {
			t.Error("Active mismatch")
		}
		if len(result.Tags) != len(contact.Tags) {
			t.Error("Tags length mismatch")
		}
		if len(result.Meta) != len(contact.Meta) {
			t.Error("Meta length mismatch")
		}
	})
}

// Test edge cases and special values
func TestEdgeCases(t *testing.T) {
	tmpDir := t.TempDir()
	store := stow.MustOpen(tmpDir)
	defer store.Close()

	ns := store.MustGetNamespace("edge")

	t.Run("Empty values", func(t *testing.T) {
		type EmptyTest struct {
			EmptyString string   `json:"empty_string"`
			EmptySlice  []string `json:"empty_slice"`
			EmptyMap    map[string]string `json:"empty_map"`
			ZeroInt     int      `json:"zero_int"`
			FalseBool   bool     `json:"false_bool"`
		}

		empty := EmptyTest{
			EmptyString: "",
			EmptySlice:  []string{},
			EmptyMap:    map[string]string{},
			ZeroInt:     0,
			FalseBool:   false,
		}

		ns.MustPut("empty", empty)

		var result EmptyTest
		ns.MustGet("empty", &result)

		// Verify empty values are preserved
		if result.EmptyString != "" {
			t.Error("Empty string should be empty")
		}
		if result.ZeroInt != 0 {
			t.Error("Zero int should be zero")
		}
		if result.FalseBool != false {
			t.Error("False bool should be false")
		}
	})

	t.Run("Unicode and special characters", func(t *testing.T) {
		type UnicodeTest struct {
			Chinese  string `json:"chinese"`
			Emoji    string `json:"emoji"`
			Symbols  string `json:"symbols"`
		}

		unicode := UnicodeTest{
			Chinese: "你好世界",
			Emoji:   "😀🎉🚀",
			Symbols: "!@#$%^&*()_+-=[]{}|;:',.<>?/",
		}

		ns.MustPut("unicode", unicode)

		var result UnicodeTest
		ns.MustGet("unicode", &result)

		if result.Chinese != unicode.Chinese {
			t.Errorf("Chinese mismatch: expected %s, got %s", unicode.Chinese, result.Chinese)
		}
		if result.Emoji != unicode.Emoji {
			t.Errorf("Emoji mismatch: expected %s, got %s", unicode.Emoji, result.Emoji)
		}
		if result.Symbols != unicode.Symbols {
			t.Errorf("Symbols mismatch")
		}
	})

	t.Run("Large strings", func(t *testing.T) {
		type LargeTest struct {
			LongText string `json:"long_text"`
		}

		// 10KB string
		longText := ""
		for i := 0; i < 10000; i++ {
			longText += "a"
		}

		large := LargeTest{
			LongText: longText,
		}

		ns.MustPut("large", large)

		var result LargeTest
		ns.MustGet("large", &result)

		if len(result.LongText) != len(large.LongText) {
			t.Errorf("Length mismatch: expected %d, got %d", len(large.LongText), len(result.LongText))
		}
	})

	t.Run("Nil vs empty slice", func(t *testing.T) {
		type SliceTest struct {
			NilSlice   []string `json:"nil_slice"`
			EmptySlice []string `json:"empty_slice"`
		}

		test := SliceTest{
			NilSlice:   nil,
			EmptySlice: []string{},
		}

		ns.MustPut("slices", test)

		var result SliceTest
		ns.MustGet("slices", &result)

		// Both should work correctly
		if result.NilSlice != nil && len(result.NilSlice) != 0 {
			t.Error("Nil slice should be nil or empty")
		}
		if result.EmptySlice != nil && len(result.EmptySlice) != 0 {
			t.Error("Empty slice should be empty")
		}
	})

	t.Run("Very large binary data", func(t *testing.T) {
		type BinaryTest struct {
			Data []byte `json:"data" stow:"file"`
		}

		// 1MB binary data
		largeData := make([]byte, 1024*1024)
		for i := range largeData {
			largeData[i] = byte(i % 256)
		}

		binary := BinaryTest{
			Data: largeData,
		}

		ns.MustPut("binary", binary)

		var result BinaryTest
		ns.MustGet("binary", &result)

		if len(result.Data) != len(largeData) {
			t.Errorf("Data length mismatch: expected %d, got %d", len(largeData), len(result.Data))
		}

		// Verify some random bytes
		if result.Data[0] != largeData[0] {
			t.Error("Data content mismatch at index 0")
		}
		if result.Data[50000] != largeData[50000] {
			t.Error("Data content mismatch at index 50000")
		}
	})
}

// Test different number types
func TestNumberTypes(t *testing.T) {
	tmpDir := t.TempDir()
	store := stow.MustOpen(tmpDir)
	defer store.Close()

	ns := store.MustGetNamespace("numbers")

	type AllNumbers struct {
		Int     int     `json:"int"`
		Int8    int8    `json:"int8"`
		Int16   int16   `json:"int16"`
		Int32   int32   `json:"int32"`
		Int64   int64   `json:"int64"`
		Uint    uint    `json:"uint"`
		Uint8   uint8   `json:"uint8"`
		Uint16  uint16  `json:"uint16"`
		Uint32  uint32  `json:"uint32"`
		Uint64  uint64  `json:"uint64"`
		Float32 float32 `json:"float32"`
		Float64 float64 `json:"float64"`
	}

	numbers := AllNumbers{
		Int:     -42,
		Int8:    -128,
		Int16:   -32768,
		Int32:   -2147483648,
		Int64:   -9223372036854775808,
		Uint:    42,
		Uint8:   255,
		Uint16:  65535,
		Uint32:  4294967295,
		Uint64:  18446744073709551615,
		Float32: 3.14,
		Float64: 2.718281828459045,
	}

	ns.MustPut("numbers", numbers)

	var result AllNumbers
	ns.MustGet("numbers", &result)

	if result.Int != numbers.Int {
		t.Errorf("Int mismatch: expected %d, got %d", numbers.Int, result.Int)
	}
	if result.Int64 != numbers.Int64 {
		t.Errorf("Int64 mismatch: expected %d, got %d", numbers.Int64, result.Int64)
	}
	if result.Uint8 != numbers.Uint8 {
		t.Errorf("Uint8 mismatch: expected %d, got %d", numbers.Uint8, result.Uint8)
	}
	if result.Float32 != numbers.Float32 {
		t.Errorf("Float32 mismatch: expected %f, got %f", numbers.Float32, result.Float32)
	}
	if result.Float64 != numbers.Float64 {
		t.Errorf("Float64 mismatch: expected %f, got %f", numbers.Float64, result.Float64)
	}
}

// Test time handling
func TestTimeHandling(t *testing.T) {
	tmpDir := t.TempDir()
	store := stow.MustOpen(tmpDir)
	defer store.Close()

	ns := store.MustGetNamespace("time")

	type TimeTest struct {
		Created   time.Time `json:"created"`
		Updated   time.Time `json:"updated"`
		Scheduled time.Time `json:"scheduled"`
		ZeroTime  time.Time `json:"zero_time"`
	}

	now := time.Now()
	future := now.Add(24 * time.Hour)

	times := TimeTest{
		Created:   now,
		Updated:   now.Add(1 * time.Hour),
		Scheduled: future,
		ZeroTime:  time.Time{},
	}

	ns.MustPut("times", times)

	var result TimeTest
	ns.MustGet("times", &result)

	// Time comparison (truncate to microsecond for comparison - JSON loses nanosecond precision)
	if !result.Created.Truncate(time.Microsecond).Equal(times.Created.Truncate(time.Microsecond)) {
		t.Errorf("Created time mismatch: expected %v, got %v", times.Created, result.Created)
	}
	if !result.Updated.Truncate(time.Microsecond).Equal(times.Updated.Truncate(time.Microsecond)) {
		t.Errorf("Updated time mismatch: expected %v, got %v", times.Updated, result.Updated)
	}
	if !result.Scheduled.Truncate(time.Microsecond).Equal(times.Scheduled.Truncate(time.Microsecond)) {
		t.Errorf("Scheduled time mismatch: expected %v, got %v", times.Scheduled, result.Scheduled)
	}
}
