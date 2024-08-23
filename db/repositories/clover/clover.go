package clover

import (
	"fmt"

	clover "github.com/ostafen/clover/v2"
)

// NewDB initializes and sets up the clover database using bbolt under the hood.
// Additionally, it automatically creates collections for the necessary types.
func NewDB(path string, collections []string) (*clover.DB, error) {
	db, err := clover.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	for _, collection := range collections {
		if err := db.CreateCollection(collection); err != nil {
			return nil, fmt.Errorf("failed to create collection %s: %w", collection, err)
		}
	}

	return db, nil
}
