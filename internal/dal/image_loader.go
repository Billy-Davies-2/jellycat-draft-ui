package dal

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
)

// LoadImagesIntoDatabase loads image files from the static/images directory into the database
func LoadImagesIntoDatabase(db *sql.DB, imagesDir string) error {
	// Get list of all PNG files in the images directory
	files, err := filepath.Glob(filepath.Join(imagesDir, "*.png"))
	if err != nil {
		return fmt.Errorf("failed to list image files: %w", err)
	}

	if len(files) == 0 {
		// No files to migrate, that's okay
		return nil
	}

	for _, filePath := range files {
		// Read the image file
		imageData, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("failed to read image file %s: %w", filePath, err)
		}

		// Extract filename without extension to match player image references
		fileName := filepath.Base(filePath)
		// The new image path format is /images/filename.png
		imageName := "/images/" + fileName

		// Update players table with image data
		result, err := db.Exec(`
			UPDATE players 
			SET image_data = $1 
			WHERE image = $2
		`, imageData, imageName)

		if err != nil {
			return fmt.Errorf("failed to update image data for %s: %w", fileName, err)
		}

		rowsAffected, _ := result.RowsAffected()
		if rowsAffected > 0 {
			fmt.Printf("Loaded image data for %s (%d bytes, %d players updated)\n", fileName, len(imageData), rowsAffected)
		}
	}

	return nil
}

// MigrateImagesToDatabase is a helper function that can be called during seed
func (p *PostgresDAL) MigrateImagesToDatabase() error {
	imagesDir := "static/images"
	if _, err := os.Stat(imagesDir); os.IsNotExist(err) {
		// Images directory doesn't exist, skip migration
		return nil
	}

	return LoadImagesIntoDatabase(p.db, imagesDir)
}

// GetPlayerImage retrieves image data for a player by ID
func (p *PostgresDAL) GetPlayerImage(playerID string) ([]byte, error) {
	var imageData []byte
	err := p.db.QueryRow(`SELECT image_data FROM players WHERE id = $1`, playerID).Scan(&imageData)
	if err != nil {
		return nil, err
	}
	return imageData, nil
}

// GetPlayerImageByPath retrieves image data by the image path
func (p *PostgresDAL) GetPlayerImageByPath(imagePath string) ([]byte, error) {
	var imageData []byte
	err := p.db.QueryRow(`SELECT image_data FROM players WHERE image = $1 LIMIT 1`, imagePath).Scan(&imageData)
	if err != nil {
		return nil, err
	}
	return imageData, nil
}
