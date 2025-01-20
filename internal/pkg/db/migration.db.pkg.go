package database

import (
	"fmt"
)

func (db *Database) RunMigrations() error {
	// Add your models here
	models := []interface{}{
		// Add more models as needed
	}

	for _, model := range models {
		if err := db.AutoMigrate(model); err != nil {
			return fmt.Errorf("failed to migrate %T: %w", model, err)
		}
	}

	return nil
}
