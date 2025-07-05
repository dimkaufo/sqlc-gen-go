# SQLC-GEN-GO Plugin Example

This example demonstrates the `sqlc-gen-go` plugin for generating Go code from SQL queries, including advanced features like **nested result grouping** for complex relational data.

## Structure

```
example/
├── sqlcin/              # SQLC input files
│   ├── sqlc.yaml        # SQLC configuration with plugin setup
│   ├── schema.sql       # Database schema (authors, books, reviews, labels)
│   └── queries/         # SQL queries organized by domain
│       ├── authors.sql  # Author-related queries with JOIN operations
│       └── books.sql    # Book-related queries
├── sqlcout/             # SQLC generated files (organized by package)
│   ├── entity/          # Generated model structs
│   │   └── model.gen.go # Database entity models
│   └── query/           # Generated query functions
│       ├── queries.gen.go         # Core query interface
│       ├── authors.sql.gen.go     # Standard author query functions
│       ├── authors_nested.sql.gen.go  # Nested grouping functions
│       └── books.sql.gen.go       # Book query functions
├── example.go           # Demo showing plugin functionality
├── go.mod              # Go module configuration
├── go.sum              # Go module checksums
└── README.md           # This file
```

## Key Features Demonstrated

### 🔹 **Multi-Package Organization**
- **Entity Package**: Clean separation of database models in `entity/model.gen.go`
- **Query Package**: All query functions organized in the `query/` directory
- **Import Path Support**: Proper cross-package imports with configurable paths

### 🔹 **Nested Result Grouping** (NEW!)
- **Automatic Grouping**: Converts flat JOIN result rows into nested Go structures
- **Multi-level Nesting**: Supports deep nesting (Author → Books → Reviews)
- **Type Safety**: Full compile-time type checking for all nested structures
- **Flexible Configuration**: YAML-based configuration for grouping behavior

### 🔹 **Advanced SQL Features**
- **Complex JOINs**: Multi-table LEFT JOINs with proper null handling
- **Embedded Results**: Uses `sqlc.embed()` for clean column organization
- **UUID Support**: Native PostgreSQL UUID support with `pgtype.UUID`
- **Proper Nullables**: Handles nullable fields with `pgtype` types

## How Nested Grouping Works

The plugin automatically generates grouping functions that transform flat SQL results into nested structures:

**Input (Flat Rows):**
```sql
-- Query with JOINs returns flat rows
SELECT a.id, a.name, sqlc.embed(b), sqlc.embed(r), sqlc.embed(l)
FROM authors a
LEFT JOIN books b ON b.author_id = a.id
LEFT JOIN reviews r ON r.book_id = b.id
LEFT JOIN labels l ON l.author_id = a.id
```

**Output (Nested Structures):**
```go
type GetAuthorsGroup struct {
    ID     pgtype.UUID
    Name   string
    Age    int32
    Books  []*GetAuthorsBook      // Nested books
    Labels []*GetAuthorsLabel     // Nested labels
}

type GetAuthorsBook struct {
    ID          pgtype.UUID
    Title       string
    AuthorID    pgtype.UUID
    Reviews     []*GetAuthorsBookReview  // Double-nested reviews
}
```

**Generated Grouping Function:**
```go
func GroupGetAuthors(rows []*GetAuthorsRow) []*GetAuthorsGroup {
    // Automatically groups flat rows into nested structure
    // Handles deduplication and proper nesting
}
```

## Configuration

The `sqlcin/sqlc.yaml` file shows advanced plugin configuration:

```yaml
version: "2"
sql:
  - engine: "postgresql"
    queries: "./queries"              # Directory-based query organization
    schema: "schema.sql"
    codegen:
      - plugin: "sqlc-gen-go"
        out: "../sqlcout"
        options:
          package: "query"
          sql_package: "pgx/v5"
          
          # Multi-package output organization
          output_query_files_directory: "query"
          output_models_file_name: "entity/model.gen.go"
          output_models_package: "entity"
          models_package_import_path: "github.com/sqlc-dev/sqlc-gen-go/example/sqlcout/entity"
          
          # Pointer configuration
          emit_result_struct_pointers: true
          emit_params_struct_pointers: true
          
          # Nested grouping configuration
          nested:
            - query: "GetAuthors"                    # Query to enhance
              struct_root: "GetAuthorsGroup"         # Root struct name
              group:
                - struct_in: "Book"                  # Input field name
                  struct_out: "GetAuthorsBook"       # Output struct name
                  nested:                            # Sub-nesting
                    - struct_in: "Review"
                      struct_out: "GetAuthorsBookReview"
                - struct_in: "Label"
                  struct_out: "GetAuthorsLabel"
```

## Running the Example

1. **Build the plugin** (from the root directory):
   ```bash
   cd ..
   make bin/sqlc-gen-go
   ```

2. **Generate SQLC code**:
   ```bash
   cd example/sqlcin
   sqlc generate
   ```

3. **Run the demo**:
   ```bash
   cd ..
   go run example.go
   ```

## Expected Output

The demo will show:
1. **Input**: Flat rows from JOIN query (showing duplicated author/book data)
2. **Processing**: Plugin-generated grouping function in action
3. **Output**: Clean nested structure with proper deduplication
4. **Summary**: Statistics showing the transformation results

```
=== SQLC-GEN-NESTED Plugin Demo ===

Input: 12 flat rows from SQLC query
----
Row 1: Author=J.K. Rowling, Book=Harry Potter and the Sorcerer's Stone, Review Rating=5
Row 2: Author=J.K. Rowling, Book=Harry Potter and the Sorcerer's Stone, Review Rating=4
Row 3: Author=J.K. Rowling, Book=Harry Potter and the Philosopher's Stone, Review Rating=5
... and 9 more rows

=== Generated Grouping Function ===

Output: 3 authors with nested books and reviews
----
Author 1: J.K. Rowling (ID: ...)
  Book 1: Harry Potter and the Sorcerer's Stone (ID: ...)
    Review 1: 5 stars by Alice (ID: ...)
    Review 2: 4 stars by Bob (ID: ...)
  Book 2: Harry Potter and the Philosopher's Stone (ID: ...)
    Review 1: 5 stars by Carol (ID: ...)
  Labels:
    Label 1: Fantasy (ID: ...)

🎉 Plugin working correctly with real SQLC generated code!
```

## Advanced Features

### ✨ **Pluralization Handling**
- Automatic proper pluralization (e.g., "Vacancy" → "Vacancies", not "Vacancys")
- Case-preserving pluralization for Go struct field names
- Uses `github.com/gobuffalo/flect` for linguistic accuracy

### ✨ **Type Safety & Performance**
- Zero reflection - all code generated at compile time
- Proper null handling with `pgtype` package
- Memory-efficient grouping algorithms
- Full IDE support with auto-completion

### ✨ **Flexible Configuration**
- Per-query grouping configuration
- Customizable struct names and relationships
- Support for both slice and single object relationships
- Configurable pointer vs value semantics

## Files Generated

- `entity/model.gen.go`: Database entity models (Author, Book, Review, Label)
- `query/queries.gen.go`: Core Queries interface and constructor
- `query/authors.sql.gen.go`: Standard SQLC query functions (GetAuthors, GetAuthor)
- `query/authors_nested.sql.gen.go`: **NEW** - Nested grouping functions (GroupGetAuthors, GroupGetAuthor)
- `query/books.sql.gen.go`: Book-related query functions

This example demonstrates how `sqlc-gen-go` bridges the gap between flat SQL results and clean, type-safe Go data structures, making it easy to work with complex relational data in Go applications.
