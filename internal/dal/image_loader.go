package dal

import (
	"database/sql"
	"fmt"
	"mime"
	"os"
	"path/filepath"
	"strings"
)

// LoadImagesIntoDatabase loads image files from the static/images directory into the database
func LoadImagesIntoDatabase(db *sql.DB, imagesDir string) error {
	entries, err := os.ReadDir(imagesDir)
	if err != nil {
		return fmt.Errorf("failed to list image files: %w", err)
	}

	if len(entries) == 0 {
		// No files to migrate, that's okay
		return nil
	}

	for _, entry := range entries {
		if entry.IsDir() || !isSupportedImageExtension(entry.Name()) {
			continue
		}

		filePath := filepath.Join(imagesDir, entry.Name())

		// Read the image file
		imageData, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("failed to read image file %s: %w", filePath, err)
		}

		fileName := filepath.Base(filePath)
		imageName := "/images/" + fileName
		contentType := contentTypeForFilename(fileName)

		if _, err := db.Exec(`
			INSERT INTO images (path, filename, content_type, data)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (path) DO UPDATE
			SET filename = EXCLUDED.filename,
				content_type = EXCLUDED.content_type,
				data = EXCLUDED.data,
				updated_at = CURRENT_TIMESTAMP
		`, imageName, fileName, contentType, imageData); err != nil {
			return fmt.Errorf("failed to save image %s: %w", fileName, err)
		}

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

// GetImageByPath retrieves user-managed image bytes and content type by URL path.
func (p *PostgresDAL) GetImageByPath(imagePath string) ([]byte, string, error) {
	var imageData []byte
	var contentType string
	err := p.db.QueryRow(`SELECT data, content_type FROM images WHERE path = $1`, imagePath).Scan(&imageData, &contentType)
	if err == nil {
		return imageData, contentType, nil
	}
	if err != sql.ErrNoRows {
		return nil, "", err
	}

	imageData, err = p.GetPlayerImageByPath(imagePath)
	if err != nil {
		return nil, "", err
	}
	return imageData, contentTypeForFilename(imagePath), nil
}

// SaveImage stores or replaces an image asset at a public /images/... path.
func (p *PostgresDAL) SaveImage(path, contentType string, data []byte) error {
	if !strings.HasPrefix(path, "/images/") {
		return fmt.Errorf("image path must start with /images/")
	}
	filename := strings.TrimPrefix(path, "/images/")
	if filename == "" {
		return fmt.Errorf("image filename is required")
	}
	if contentType == "" {
		contentType = contentTypeForFilename(filename)
	}

	_, err := p.db.Exec(`
		INSERT INTO images (path, filename, content_type, data)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (path) DO UPDATE
		SET filename = EXCLUDED.filename,
			content_type = EXCLUDED.content_type,
			data = EXCLUDED.data,
			updated_at = CURRENT_TIMESTAMP
	`, path, filename, contentType, data)
	return err
}

// ListImages returns public paths for image assets stored in Postgres.
func (p *PostgresDAL) ListImages() ([]string, error) {
	rows, err := p.db.Query(`SELECT path FROM images ORDER BY filename`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	images := []string{}
	for rows.Next() {
		var path string
		if err := rows.Scan(&path); err != nil {
			return nil, err
		}
		images = append(images, path)
	}
	return images, rows.Err()
}

func isSupportedImageExtension(filename string) bool {
	switch strings.ToLower(filepath.Ext(filename)) {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp":
		return true
	default:
		return false
	}
}

func contentTypeForFilename(filename string) string {
	contentType := mime.TypeByExtension(strings.ToLower(filepath.Ext(filename)))
	if contentType == "" {
		return "application/octet-stream"
	}
	return contentType
}
