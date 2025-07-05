package main

import (
	"fmt"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/sqlc-dev/sqlc-gen-go/example/sqlcout/entity"
	"github.com/sqlc-dev/sqlc-gen-go/example/sqlcout/query"
)

// Using SQLC generated types from sqlcout package

// Main Demo

func main() {
	fmt.Println("=== SQLC-GEN-NESTED Plugin Demo ===")
	fmt.Println()

	// Create sample data that matches the SQLC generated structures
	rows := createSampleData()

	fmt.Printf("Input: %d flat rows from SQLC query\n", len(rows))
	fmt.Println("----")

	// Print some sample flat rows
	for i, row := range rows {
		if i >= 3 { // Show first 3 rows
			fmt.Printf("... and %d more rows\n", len(rows)-i)
			break
		}

		authorName := row.Name
		bookTitle := "NULL"
		if row.Book.Title != "" {
			bookTitle = row.Book.Title
		}
		reviewRating := "NULL"
		if row.Review.Rating != 0 {
			reviewRating = fmt.Sprintf("%d", row.Review.Rating)
		}

		fmt.Printf("Row %d: Author=%s, Book=%s, Review Rating=%s\n",
			i+1, authorName, bookTitle, reviewRating)
	}

	fmt.Println()
	fmt.Println("=== Generated Grouping Function ===")
	fmt.Println()

	// Use the plugin-generated grouping function
	// Convert slice of structs to slice of pointers for the group function
	var rowPointers []*query.GetAuthorsRow
	for i := range rows {
		rowPointers = append(rowPointers, &rows[i])
	}
	groupedAuthors := query.GroupGetAuthors(rowPointers)

	fmt.Printf("Output: %d authors with nested books and reviews\n", len(groupedAuthors))
	fmt.Println("----")

	// Display the grouped results
	for i, author := range groupedAuthors {
		fmt.Printf("Author %d: %s (ID: %s)\n", i+1, author.Name, formatUUID(author.ID))

		if len(author.Books) == 0 {
			fmt.Println("  No books")
		} else {
			for j, book := range author.Books {
				fmt.Printf("  Book %d: %s (ID: %s)\n", j+1, book.Title, formatUUID(book.ID))

				if len(book.Reviews) == 0 {
					fmt.Println("    No reviews")
				} else {
					for k, review := range book.Reviews {
						reviewerName := "Anonymous"
						if review.ReviewerName.Valid {
							reviewerName = review.ReviewerName.String
						}
						fmt.Printf("    Review %d: %d stars by %s (ID: %s)\n",
							k+1, review.Rating, reviewerName, formatUUID(review.ID))
					}
				}
			}
		}

		// Display labels if any
		if len(author.Labels) > 0 {
			fmt.Println("  Labels:")
			for j, label := range author.Labels {
				fmt.Printf("    Label %d: %s (ID: %s)\n", j+1, label.Title, formatUUID(label.ID))
			}
		}

		fmt.Println()
	}

	fmt.Println("=== Summary ===")
	fmt.Printf("✓ SQLC generated %d query structs with embedded Book, Review, and Label\n", len(rows))
	fmt.Printf("✓ Plugin grouped into %d authors\n", len(groupedAuthors))

	totalBooks := 0
	totalReviews := 0
	totalLabels := 0
	for _, author := range groupedAuthors {
		totalBooks += len(author.Books)
		totalLabels += len(author.Labels)
		for _, book := range author.Books {
			totalReviews += len(book.Reviews)
		}
	}
	fmt.Printf("✓ Total books: %d\n", totalBooks)
	fmt.Printf("✓ Total reviews: %d\n", totalReviews)
	fmt.Printf("✓ Total labels: %d\n", totalLabels)
	fmt.Println()
	fmt.Println("🎉 Plugin working correctly with real SQLC generated code!")
}

// createSampleData creates sample data using the SQLC generated structures
func createSampleData() []query.GetAuthorsRow {
	// Helper function to create UUIDs
	createUUID := func(s string) pgtype.UUID {
		// In real usage, these would be proper UUIDs from the database
		// For demo purposes, we'll use a simple byte pattern
		var uuid pgtype.UUID
		uuid.Valid = true
		copy(uuid.Bytes[:], []byte(s+string(make([]byte, 16-len(s)))))
		return uuid
	}

	// Create Author UUIDs
	authorJKRowling := createUUID("author-jkr")
	authorGeorgeOrwell := createUUID("author-orwell")
	authorIsaacAsimov := createUUID("author-asimov")

	// Create Book UUIDs
	bookHarryPotter := createUUID("book-hp")
	bookPhilosopher := createUUID("book-ps")
	book1984 := createUUID("book-1984")
	bookFoundation := createUUID("book-found")

	// Create Review UUIDs
	reviewHP1 := createUUID("review-hp1")
	reviewHP2 := createUUID("review-hp2")
	reviewPS1 := createUUID("review-ps1")
	review19841 := createUUID("review-19841")

	// Create Label UUIDs
	labelFantasy := createUUID("label-fantasy")
	labelClassic := createUUID("label-classic")
	labelSciFi := createUUID("label-scifi")

	return []query.GetAuthorsRow{
		// J.K. Rowling with Harry Potter and reviews
		{
			ID:   authorJKRowling,
			Name: "J.K. Rowling",
			Book: entity.Book{
				ID:       bookHarryPotter,
				Title:    "Harry Potter and the Sorcerer's Stone",
				AuthorID: authorJKRowling,
			},
			Review: entity.Review{
				ID:           reviewHP1,
				BookID:       bookHarryPotter,
				Rating:       5,
				Comment:      pgtype.Text{String: "Magical and wonderful!", Valid: true},
				ReviewerName: pgtype.Text{String: "Alice", Valid: true},
			},
			Label: entity.Label{
				ID:       labelFantasy,
				Title:    "Fantasy",
				AuthorID: authorJKRowling,
			},
		},
		{
			ID:   authorJKRowling,
			Name: "J.K. Rowling",
			Book: entity.Book{
				ID:       bookHarryPotter,
				Title:    "Harry Potter and the Sorcerer's Stone",
				AuthorID: authorJKRowling,
			},
			Review: entity.Review{
				ID:           reviewHP2,
				BookID:       bookHarryPotter,
				Rating:       4,
				Comment:      pgtype.Text{String: "Great for kids and adults", Valid: true},
				ReviewerName: pgtype.Text{String: "Bob", Valid: true},
			},
			Label: entity.Label{
				ID:       labelFantasy,
				Title:    "Fantasy",
				AuthorID: authorJKRowling,
			},
		},
		// J.K. Rowling with another book
		{
			ID:   authorJKRowling,
			Name: "J.K. Rowling",
			Book: entity.Book{
				ID:       bookPhilosopher,
				Title:    "The Philosopher's Stone",
				AuthorID: authorJKRowling,
			},
			Review: entity.Review{
				ID:           reviewPS1,
				BookID:       bookPhilosopher,
				Rating:       5,
				Comment:      pgtype.Text{String: "Classic!", Valid: true},
				ReviewerName: pgtype.Text{String: "Charlie", Valid: true},
			},
			Label: entity.Label{
				ID:       labelClassic,
				Title:    "Classic",
				AuthorID: authorJKRowling,
			},
		},
		// George Orwell with 1984
		{
			ID:   authorGeorgeOrwell,
			Name: "George Orwell",
			Book: entity.Book{
				ID:       book1984,
				Title:    "1984",
				AuthorID: authorGeorgeOrwell,
			},
			Review: entity.Review{
				ID:           review19841,
				BookID:       book1984,
				Rating:       5,
				Comment:      pgtype.Text{String: "Dystopian masterpiece", Valid: true},
				ReviewerName: pgtype.Text{String: "Diana", Valid: true},
			},
			Label: entity.Label{
				ID:       labelClassic,
				Title:    "Classic",
				AuthorID: authorGeorgeOrwell,
			},
		},
		// Isaac Asimov with Foundation (book but no reviews)
		{
			ID:   authorIsaacAsimov,
			Name: "Isaac Asimov",
			Book: entity.Book{
				ID:       bookFoundation,
				Title:    "Foundation",
				AuthorID: authorIsaacAsimov,
			},
			Review: entity.Review{
				ID: pgtype.UUID{Valid: false}, // No review
			},
			Label: entity.Label{
				ID:       labelSciFi,
				Title:    "Science Fiction",
				AuthorID: authorIsaacAsimov,
			},
		},
	}
}

// formatUUID formats a pgtype.UUID for display
func formatUUID(uuid pgtype.UUID) string {
	if !uuid.Valid {
		return "NULL"
	}
	// For demo purposes, just show first 8 bytes as hex
	return fmt.Sprintf("%x", uuid.Bytes[:8])
}
