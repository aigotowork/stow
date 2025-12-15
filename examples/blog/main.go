package main

import (
	"fmt"
	"log"
	"time"

	"github.com/aigotowork/stow"
)

// Author represents a blog author
type Author struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	Bio   string `json:"bio"`
}

// Comment represents a blog comment
type Comment struct {
	ID        string    `json:"id"`
	Author    string    `json:"author"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

// BlogPost represents a blog post with nested structures
type BlogPost struct {
	Title       string    `json:"title"`
	Slug        string    `json:"slug"`
	Author      Author    `json:"author"` // Nested struct
	Content     string    `json:"content"`
	Summary     string    `json:"summary"`
	Tags        []string  `json:"tags"`
	Comments    []Comment `json:"comments"` // Array of nested structs
	ViewCount   int       `json:"view_count"`
	IsPublished bool      `json:"is_published"`
	PublishedAt time.Time `json:"published_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Category represents a blog category with nested posts
type Category struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	PostSlugs   []string `json:"post_slugs"` // References to posts
	PostCount   int      `json:"post_count"`
}

// BlogStats represents overall blog statistics
type BlogStats struct {
	TotalPosts     int       `json:"total_posts"`
	TotalComments  int       `json:"total_comments"`
	TotalViews     int       `json:"total_views"`
	LastUpdated    time.Time `json:"last_updated"`
	PopularTags    []string  `json:"popular_tags"`
	RecentActivity []string  `json:"recent_activity"`
}

func main() {
	storePath := "./data/blog_example"

	store, err := stow.Open(storePath)
	if err != nil {
		log.Fatal(err)
	}
	defer store.Close()

	fmt.Println("=== Blog Management System ===")
	fmt.Println()

	// Example 1: Create and store blog posts
	demonstrateBlogPosts(store)

	// Example 2: Manage categories
	demonstrateCategories(store)

	// Example 3: Update posts with comments
	demonstrateComments(store)

	// Example 4: Blog statistics
	demonstrateStatistics(store)

	fmt.Println("\n=== Blog system ready! ===")
	fmt.Printf("\nData stored at: %s\n", storePath)
	fmt.Println("Namespaces:")
	fmt.Println("  - posts/     : Blog posts with nested author and comments")
	fmt.Println("  - categories/: Post categories")
	fmt.Println("  - stats/     : Blog statistics")}

func demonstrateBlogPosts(store stow.Store) {
	fmt.Println("1. Creating Blog Posts")

	ns, err := store.GetNamespace("posts")
	if err != nil {
		log.Fatal(err)
	}

	// Create sample posts
	posts := []BlogPost{
		{
			Title: "Getting Started with Stow",
			Slug:  "getting-started-with-stow",
			Author: Author{
				Name:  "Alice Johnson",
				Email: "alice@example.com",
				Bio:   "Software engineer passionate about simple solutions",
			},
			Content: `
# Getting Started with Stow

Stow is a lightweight embedded key-value store designed for simplicity and transparency.
In this post, we'll explore the basics of using Stow in your Go applications.

## Installation

First, install Stow using go get:

` + "```bash" + `
go get github.com/aigotowork/stow
` + "```" + `

## Basic Usage

Here's a simple example...
`,
			Summary:     "Learn the basics of using Stow in your Go applications",
			Tags:        []string{"tutorial", "golang", "storage"},
			Comments:    []Comment{},
			ViewCount:   0,
			IsPublished: true,
			PublishedAt: time.Now().Add(-24 * time.Hour),
			UpdatedAt:   time.Now().Add(-24 * time.Hour),
		},
		{
			Title: "Advanced Stow Features",
			Slug:  "advanced-stow-features",
			Author: Author{
				Name:  "Bob Smith",
				Email: "bob@example.com",
				Bio:   "Database enthusiast and performance optimizer",
			},
			Content: `
# Advanced Features in Stow

Now that you're familiar with the basics, let's dive into some advanced features
that make Stow a powerful tool for embedded storage.

## Nested Struct Support

Stow now supports nested structs, making it easy to store complex data...

## Async Compaction

Keep your storage optimized with async compaction...

## Concurrent Operations

Leverage key-level locking for better concurrency...
`,
			Summary:     "Explore advanced features like nested structs and async operations",
			Tags:        []string{"advanced", "golang", "performance"},
			Comments:    []Comment{},
			ViewCount:   0,
			IsPublished: true,
			PublishedAt: time.Now().Add(-12 * time.Hour),
			UpdatedAt:   time.Now().Add(-12 * time.Hour),
		},
		{
			Title: "Stow vs Traditional Databases",
			Slug:  "stow-vs-databases",
			Author: Author{
				Name:  "Alice Johnson",
				Email: "alice@example.com",
				Bio:   "Software engineer passionate about simple solutions",
			},
			Content: `
# Stow vs Traditional Databases

When should you use Stow instead of a full-featured database?
This post explores the trade-offs and use cases...

## When to Use Stow

- Configuration management
- Small to medium datasets
- Need for human-readable storage
- Embedded applications

## When NOT to Use Stow

- High concurrency requirements
- Complex queries
- Large datasets (> 100MB)
- Transactions across multiple keys
`,
			Summary:     "Understanding when to use Stow vs traditional databases",
			Tags:        []string{"comparison", "architecture", "databases"},
			Comments:    []Comment{},
			ViewCount:   0,
			IsPublished: false, // Draft
			PublishedAt: time.Time{},
			UpdatedAt:   time.Now(),
		},
	}

	for _, post := range posts {
		if err := ns.Put(post.Slug, post); err != nil {
			log.Fatal(err)
		}
		status := "published"
		if !post.IsPublished {
			status = "draft"
		}
		fmt.Printf("   ✓ Created post: '%s' (%s)\n", post.Title, status)
		fmt.Printf("     - Author: %s\n", post.Author.Name)
		fmt.Printf("     - Tags: %v\n", post.Tags)
	}
	fmt.Println()
}

func demonstrateCategories(store stow.Store) {
	fmt.Println("2. Managing Categories")

	ns, err := store.GetNamespace("categories")
	if err != nil {
		log.Fatal(err)
	}

	categories := []Category{
		{
			Name:        "Tutorials",
			Description: "Step-by-step guides and tutorials",
			PostSlugs:   []string{"getting-started-with-stow"},
			PostCount:   1,
		},
		{
			Name:        "Advanced Topics",
			Description: "Deep dives into advanced features",
			PostSlugs:   []string{"advanced-stow-features"},
			PostCount:   1,
		},
		{
			Name:        "Comparisons",
			Description: "Comparing Stow with other solutions",
			PostSlugs:   []string{"stow-vs-databases"},
			PostCount:   1,
		},
	}

	for _, cat := range categories {
		if err := ns.Put(cat.Name, cat); err != nil {
			log.Fatal(err)
		}
		fmt.Printf("   ✓ Created category: '%s' (%d posts)\n", cat.Name, cat.PostCount)
	}
	fmt.Println()
}

func demonstrateComments(store stow.Store) {
	fmt.Println("3. Adding Comments to Posts")

	ns, err := store.GetNamespace("posts")
	if err != nil {
		log.Fatal(err)
	}

	// Retrieve a post
	var post BlogPost
	if err := ns.Get("getting-started-with-stow", &post); err != nil {
		log.Fatal(err)
	}

	// Add comments
	post.Comments = []Comment{
		{
			ID:        "c1",
			Author:    "Reader One",
			Content:   "Great introduction! Very clear and helpful.",
			CreatedAt: time.Now().Add(-2 * time.Hour),
		},
		{
			ID:        "c2",
			Author:    "Developer",
			Content:   "Thanks for this tutorial. One question: how does it handle concurrency?",
			CreatedAt: time.Now().Add(-1 * time.Hour),
		},
		{
			ID:        "c3",
			Author:    "Alice Johnson",
			Content:   "@Developer: Great question! Stow uses key-level locking for safe concurrent access.",
			CreatedAt: time.Now().Add(-30 * time.Minute),
		},
	}

	// Increment view count
	post.ViewCount = 42
	post.UpdatedAt = time.Now()

	// Update the post
	if err := ns.Put("getting-started-with-stow", post); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("   ✓ Added %d comments to '%s'\n", len(post.Comments), post.Title)
	fmt.Printf("   ✓ Updated view count: %d\n", post.ViewCount)

	// Retrieve and display with comments
	var updated BlogPost
	if err := ns.Get("getting-started-with-stow", &updated); err != nil {
		log.Fatal(err)
	}

	fmt.Println("   ✓ Post retrieved with nested data:")
	fmt.Printf("     - Title: %s\n", updated.Title)
	fmt.Printf("     - Author: %s (%s)\n", updated.Author.Name, updated.Author.Email)
	fmt.Printf("     - Comments: %d\n", len(updated.Comments))
	for i, c := range updated.Comments {
		fmt.Printf("       %d. %s: %s\n", i+1, c.Author, c.Content)
	}
	fmt.Println()
}

func demonstrateStatistics(store stow.Store) {
	fmt.Println("4. Blog Statistics")

	ns, err := store.GetNamespace("stats")
	if err != nil {
		log.Fatal(err)
	}

	stats := BlogStats{
		TotalPosts:    3,
		TotalComments: 3,
		TotalViews:    42,
		LastUpdated:   time.Now(),
		PopularTags:   []string{"tutorial", "golang", "advanced"},
		RecentActivity: []string{
			"Added 3 comments to 'Getting Started with Stow'",
			"Published 'Advanced Stow Features'",
			"Created draft 'Stow vs Databases'",
		},
	}

	if err := ns.Put("overall", stats); err != nil {
		log.Fatal(err)
	}

	fmt.Println("   ✓ Stored blog statistics:")
	fmt.Printf("     - Total posts: %d\n", stats.TotalPosts)
	fmt.Printf("     - Total comments: %d\n", stats.TotalComments)
	fmt.Printf("     - Total views: %d\n", stats.TotalViews)
	fmt.Printf("     - Popular tags: %v\n", stats.PopularTags)

	// Retrieve stats
	var retrieved BlogStats
	if err := ns.Get("overall", &retrieved); err != nil {
		log.Fatal(err)
	}

	fmt.Println("   ✓ Statistics retrieved successfully")
	fmt.Println("   ✓ Recent activity:")
	for i, activity := range retrieved.RecentActivity {
		fmt.Printf("     %d. %s\n", i+1, activity)
	}
	fmt.Println()
}
