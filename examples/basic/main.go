package main

import (
	"fmt"
	"log"

	"github.com/aigotowork/stow"
)

func main() {
	// Create a temporary directory for testing
	storePath := "./data/example_basic"

	// Open or create store
	store, err := stow.Open(storePath)
	if err != nil {
		log.Fatal(err)
	}
	defer store.Close()

	// Get or create namespace
	ns, err := store.GetNamespace("config")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("=== Basic KV Operations ===")

	// 1. Put: Store simple data
	fmt.Println("1. Storing server configuration...")
	serverConfig := map[string]interface{}{
		"host": "localhost",
		"port": 8080,
		"ssl":  true,
	}

	if err := ns.Put("server", serverConfig); err != nil {
		log.Fatal(err)
	}
	fmt.Println("   ✓ Stored successfully")

	// 2. Get: Retrieve data
	fmt.Println("\n2. Retrieving server configuration...")
	var retrieved map[string]interface{}
	if err := ns.Get("server", &retrieved); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("   ✓ Retrieved: %+v\n", retrieved)

	// 3. Exists: Check if key exists
	fmt.Println("\n3. Checking if 'server' exists...")
	exists := ns.Exists("server")
	fmt.Printf("   ✓ Exists: %v\n", exists)

	// 4. Put: Update with new version
	fmt.Println("\n4. Updating server configuration (changing port)...")
	serverConfig["port"] = 9090
	if err := ns.Put("server", serverConfig); err != nil {
		log.Fatal(err)
	}
	fmt.Println("   ✓ Updated successfully")

	// 5. Get: Retrieve updated data
	fmt.Println("\n5. Retrieving updated configuration...")
	if err := ns.Get("server", &retrieved); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("   ✓ Retrieved: %+v\n", retrieved)

	// 6. GetHistory: View version history
	fmt.Println("\n6. Viewing version history...")
	history, err := ns.GetHistory("server")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("   ✓ Total versions: %d\n", len(history))
	for _, v := range history {
		fmt.Printf("     - Version %d: %s at %s\n", v.Version, v.Operation, v.Timestamp.Format("15:04:05"))
	}

	// 7. List: List all keys
	fmt.Println("\n7. Listing all keys...")
	keys, err := ns.List()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("   ✓ Keys: %v\n", keys)

	// 8. Delete: Mark key as deleted
	fmt.Println("\n8. Deleting 'server' key...")
	if err := ns.Delete("server"); err != nil {
		log.Fatal(err)
	}
	fmt.Println("   ✓ Deleted successfully")

	// 9. Exists: Verify deletion
	fmt.Println("\n9. Checking if 'server' still exists...")
	exists = ns.Exists("server")
	fmt.Printf("   ✓ Exists: %v\n", exists)

	// 10. Stats: Get namespace statistics
	fmt.Println("\n10. Getting namespace statistics...")
	stats, err := ns.Stats()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("   ✓ Statistics:\n")
	fmt.Printf("     - Key count: %d\n", stats.KeyCount)
	fmt.Printf("     - Blob count: %d\n", stats.BlobCount)
	fmt.Printf("     - Total size: %d bytes\n", stats.TotalSize)

	fmt.Println("\n=== Example completed successfully! ===")
	fmt.Printf("\nYou can inspect the data at: %s/config/\n", storePath)
	fmt.Println("The JSONL files are human-readable and can be edited directly.")
}
