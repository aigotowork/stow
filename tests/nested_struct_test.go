package stow_test

import (
	"testing"

	"github.com/aigotowork/stow"
)

// Test nested struct support

type Address struct {
	Street string `json:"street"`
	City   string `json:"city"`
	Zip    int    `json:"zip"`
}

type Person struct {
	Name    string  `json:"name"`
	Age     int     `json:"age"`
	Address Address `json:"address"`
}

type Company struct {
	Name      string   `json:"name"`
	Address   Address  `json:"address"`
	CEO       Person   `json:"ceo"`
	Employees []Person `json:"employees,omitempty"`
}

type Node struct {
	Value    string `json:"value"`
	Parent   *Node  `json:"parent,omitempty"`
	Children []Node `json:"children,omitempty"`
}

func TestNestedStructBasic(t *testing.T) {
	tmpDir := t.TempDir()
	store := stow.MustOpen(tmpDir)
	defer store.Close()

	ns := store.MustGetNamespace("test")

	// Create a person with nested address
	person := Person{
		Name: "John Doe",
		Age:  30,
		Address: Address{
			Street: "123 Main St",
			City:   "Springfield",
			Zip:    12345,
		},
	}

	// Store the person
	err := ns.Put("person1", person)
	if err != nil {
		t.Fatalf("Failed to put person: %v", err)
	}

	// Retrieve the person
	var retrieved Person
	err = ns.Get("person1", &retrieved)
	if err != nil {
		t.Fatalf("Failed to get person: %v", err)
	}

	// Verify all fields including nested struct
	if retrieved.Name != person.Name {
		t.Errorf("Name mismatch: expected %s, got %s", person.Name, retrieved.Name)
	}
	if retrieved.Age != person.Age {
		t.Errorf("Age mismatch: expected %d, got %d", person.Age, retrieved.Age)
	}
	if retrieved.Address.Street != person.Address.Street {
		t.Errorf("Address.Street mismatch: expected %s, got %s", person.Address.Street, retrieved.Address.Street)
	}
	if retrieved.Address.City != person.Address.City {
		t.Errorf("Address.City mismatch: expected %s, got %s", person.Address.City, retrieved.Address.City)
	}
	if retrieved.Address.Zip != person.Address.Zip {
		t.Errorf("Address.Zip mismatch: expected %d, got %d", person.Address.Zip, retrieved.Address.Zip)
	}
}

func TestNestedStructMultipleLevels(t *testing.T) {
	tmpDir := t.TempDir()
	store := stow.MustOpen(tmpDir)
	defer store.Close()

	ns := store.MustGetNamespace("test")

	// Create a company with multiple levels of nesting
	company := Company{
		Name: "Acme Corp",
		Address: Address{
			Street: "456 Corporate Blvd",
			City:   "New York",
			Zip:    10001,
		},
		CEO: Person{
			Name: "Jane Smith",
			Age:  45,
			Address: Address{
				Street: "789 Executive Ave",
				City:   "Manhattan",
				Zip:    10002,
			},
		},
		Employees: []Person{
			{
				Name: "Bob Johnson",
				Age:  28,
				Address: Address{
					Street: "321 Worker St",
					City:   "Brooklyn",
					Zip:    11201,
				},
			},
			{
				Name: "Alice Williams",
				Age:  32,
				Address: Address{
					Street: "654 Employee Rd",
					City:   "Queens",
					Zip:    11101,
				},
			},
		},
	}

	// Store the company
	err := ns.Put("company1", company)
	if err != nil {
		t.Fatalf("Failed to put company: %v", err)
	}

	// Retrieve the company
	var retrieved Company
	err = ns.Get("company1", &retrieved)
	if err != nil {
		t.Fatalf("Failed to get company: %v", err)
	}

	// Verify company fields
	if retrieved.Name != company.Name {
		t.Errorf("Company name mismatch: expected %s, got %s", company.Name, retrieved.Name)
	}

	// Verify company address
	if retrieved.Address.City != company.Address.City {
		t.Errorf("Company city mismatch: expected %s, got %s", company.Address.City, retrieved.Address.City)
	}

	// Verify CEO nested struct
	if retrieved.CEO.Name != company.CEO.Name {
		t.Errorf("CEO name mismatch: expected %s, got %s", company.CEO.Name, retrieved.CEO.Name)
	}
	if retrieved.CEO.Address.City != company.CEO.Address.City {
		t.Errorf("CEO city mismatch: expected %s, got %s", company.CEO.Address.City, retrieved.CEO.Address.City)
	}

	// Verify employees array
	if len(retrieved.Employees) != len(company.Employees) {
		t.Fatalf("Employees count mismatch: expected %d, got %d", len(company.Employees), len(retrieved.Employees))
	}

	for i, emp := range company.Employees {
		if retrieved.Employees[i].Name != emp.Name {
			t.Errorf("Employee[%d] name mismatch: expected %s, got %s", i, emp.Name, retrieved.Employees[i].Name)
		}
		if retrieved.Employees[i].Address.City != emp.Address.City {
			t.Errorf("Employee[%d] city mismatch: expected %s, got %s", i, emp.Address.City, retrieved.Employees[i].Address.City)
		}
	}
}

func TestNestedStructWithPointers(t *testing.T) {
	tmpDir := t.TempDir()
	store := stow.MustOpen(tmpDir)
	defer store.Close()

	ns := store.MustGetNamespace("test")

	// Create a tree structure with pointer references
	root := Node{
		Value: "root",
		Children: []Node{
			{
				Value: "child1",
			},
			{
				Value: "child2",
			},
		},
	}

	// Store the node
	err := ns.Put("tree1", root)
	if err != nil {
		t.Fatalf("Failed to put tree: %v", err)
	}

	// Retrieve the node
	var retrieved Node
	err = ns.Get("tree1", &retrieved)
	if err != nil {
		t.Fatalf("Failed to get tree: %v", err)
	}

	// Verify structure
	if retrieved.Value != root.Value {
		t.Errorf("Root value mismatch: expected %s, got %s", root.Value, retrieved.Value)
	}

	if len(retrieved.Children) != len(root.Children) {
		t.Fatalf("Children count mismatch: expected %d, got %d", len(root.Children), len(retrieved.Children))
	}

	for i, child := range root.Children {
		if retrieved.Children[i].Value != child.Value {
			t.Errorf("Child[%d] value mismatch: expected %s, got %s", i, child.Value, retrieved.Children[i].Value)
		}
	}
}

func TestNestedStructRoundTrip(t *testing.T) {
	tmpDir := t.TempDir()
	store := stow.MustOpen(tmpDir)
	defer store.Close()

	ns := store.MustGetNamespace("test")

	// Test multiple round trips
	original := Person{
		Name: "Test Person",
		Age:  25,
		Address: Address{
			Street: "100 Test Ave",
			City:   "Test City",
			Zip:    99999,
		},
	}

	// First write
	err := ns.Put("roundtrip", original)
	if err != nil {
		t.Fatalf("First Put failed: %v", err)
	}

	// First read
	var first Person
	err = ns.Get("roundtrip", &first)
	if err != nil {
		t.Fatalf("First Get failed: %v", err)
	}

	// Modify and write again
	first.Age = 26
	first.Address.Street = "200 Test Ave"
	err = ns.Put("roundtrip", first)
	if err != nil {
		t.Fatalf("Second Put failed: %v", err)
	}

	// Second read
	var second Person
	err = ns.Get("roundtrip", &second)
	if err != nil {
		t.Fatalf("Second Get failed: %v", err)
	}

	// Verify modifications
	if second.Age != 26 {
		t.Errorf("Age not updated: expected 26, got %d", second.Age)
	}
	if second.Address.Street != "200 Test Ave" {
		t.Errorf("Street not updated: expected '200 Test Ave', got %s", second.Address.Street)
	}
	if second.Name != original.Name {
		t.Errorf("Name should remain: expected %s, got %s", original.Name, second.Name)
	}
}

func TestNestedStructWithMap(t *testing.T) {
	tmpDir := t.TempDir()
	store := stow.MustOpen(tmpDir)
	defer store.Close()

	ns := store.MustGetNamespace("test")

	// Test nested struct mixed with map
	data := map[string]interface{}{
		"id":   "123",
		"name": "Mixed Test",
		"person": Person{
			Name: "Embedded Person",
			Age:  40,
			Address: Address{
				Street: "999 Mixed St",
				City:   "Mixed City",
				Zip:    88888,
			},
		},
	}

	// Store the mixed data
	err := ns.Put("mixed1", data)
	if err != nil {
		t.Fatalf("Failed to put mixed data: %v", err)
	}

	// Retrieve as map
	var retrieved map[string]interface{}
	err = ns.Get("mixed1", &retrieved)
	if err != nil {
		t.Fatalf("Failed to get mixed data: %v", err)
	}

	// Verify basic fields
	if retrieved["id"] != "123" {
		t.Errorf("ID mismatch: expected '123', got %v", retrieved["id"])
	}
	if retrieved["name"] != "Mixed Test" {
		t.Errorf("Name mismatch: expected 'Mixed Test', got %v", retrieved["name"])
	}

	// Verify nested person struct (could be either struct or map after round trip)
	personValue := retrieved["person"]

	// Try as map first
	if personMap, ok := personValue.(map[string]interface{}); ok {
		if personMap["name"] != "Embedded Person" {
			t.Errorf("Person name mismatch: expected 'Embedded Person', got %v", personMap["name"])
		}

		// Verify nested address within person
		address, ok := personMap["address"].(map[string]interface{})
		if !ok {
			t.Fatalf("Address field is not a map: %T", personMap["address"])
		}

		if address["city"] != "Mixed City" {
			t.Errorf("Address city mismatch: expected 'Mixed City', got %v", address["city"])
		}
	} else if personStruct, ok := personValue.(Person); ok {
		// If it's preserved as a struct (which our implementation does)
		if personStruct.Name != "Embedded Person" {
			t.Errorf("Person name mismatch: expected 'Embedded Person', got %v", personStruct.Name)
		}
		if personStruct.Address.City != "Mixed City" {
			t.Errorf("Address city mismatch: expected 'Mixed City', got %v", personStruct.Address.City)
		}
	} else {
		t.Fatalf("Person field has unexpected type: %T", personValue)
	}
}

func TestNestedStructEmpty(t *testing.T) {
	tmpDir := t.TempDir()
	store := stow.MustOpen(tmpDir)
	defer store.Close()

	ns := store.MustGetNamespace("test")

	// Test with empty nested structs
	person := Person{
		Name:    "Empty Test",
		Age:     0,
		Address: Address{}, // Empty nested struct
	}

	err := ns.Put("empty1", person)
	if err != nil {
		t.Fatalf("Failed to put person with empty address: %v", err)
	}

	var retrieved Person
	err = ns.Get("empty1", &retrieved)
	if err != nil {
		t.Fatalf("Failed to get person with empty address: %v", err)
	}

	// Verify empty address fields are zero values
	if retrieved.Address.Street != "" {
		t.Errorf("Expected empty street, got %s", retrieved.Address.Street)
	}
	if retrieved.Address.City != "" {
		t.Errorf("Expected empty city, got %s", retrieved.Address.City)
	}
	if retrieved.Address.Zip != 0 {
		t.Errorf("Expected zero zip, got %d", retrieved.Address.Zip)
	}
}
