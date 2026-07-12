package controller

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"backend/db"
	"backend/models"

	"github.com/cloudinary/cloudinary-go/v2"
	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

// UploadImage handles image uploads for a store owner.
// Expected form field: "image" (multipart/form-data).
func UploadImage(c fiber.Ctx) error {
	// Ensure request is from an authenticated store owner (middleware applied).
	userIDValue := c.Locals("userID")
	userIDString, ok := userIDValue.(string)
	if !ok || userIDString == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"message": "User not found in token",
		})
	}

	userID, err := uuid.Parse(userIDString)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid user ID",
		})
	}

	// Fetch store belonging to the user.
	var store models.Store
	if err := db.DB.Where("user_id = ?", userID).First(&store).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"message": "Store not found for this user",
		})
	}

	// Retrieve file from form.
	fileHeader, err := c.FormFile("image")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Image file is required",
		})
	}

	// Size limit – default 5 MB, can be overridden via env var later.
	const maxSize = 5 << 20 // 5 MB
	if fileHeader.Size > maxSize {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": fmt.Sprintf("File exceeds max size of %d bytes", maxSize),
		})
	}

	// Validate MIME type.
	allowedMimes := map[string]bool{"image/jpeg": true, "image/png": true, "image/webp": true}
	mime := fileHeader.Header.Get("Content-Type")
	if !allowedMimes[mime] {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Unsupported image format",
		})
	}

	// Try Cloudinary first if configured
	var cloudinaryUrl string
	cld, err := cloudinary.New()
	if err == nil {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		folderName := fmt.Sprintf("kadaitheru/%s", store.Slug)
		safeName := fmt.Sprintf("%s_%s", uuid.New().String(), strings.TrimSuffix(filepath.Base(fileHeader.Filename), filepath.Ext(fileHeader.Filename)))
		
		file, err := fileHeader.Open()
		if err == nil {
			defer file.Close()
			uploadResult, uploadErr := cld.Upload.Upload(ctx, file, uploader.UploadParams{
				Folder:   folderName,
				PublicID: safeName,
			})

			if uploadErr == nil && uploadResult.Error.Message == "" {
				cloudinaryUrl = uploadResult.SecureURL
			} else {
				fmt.Printf("Cloudinary upload failed (falling back to local): %v\n", uploadResult.Error.Message)
			}
		}
	}

	var finalUrl string

	// If Cloudinary succeeded, use that URL. Otherwise, save locally.
	if cloudinaryUrl != "" {
		finalUrl = cloudinaryUrl
	} else {
		// Local Fallback
		uploadDir := "./uploads/store_" + store.Slug
		if err := os.MkdirAll(uploadDir, 0755); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"success": false,
				"message": "Could not create upload directory",
				"error":   err.Error(),
			})
		}

		safeFileName := fmt.Sprintf("%s%s", uuid.New().String(), filepath.Ext(fileHeader.Filename))
		savePath := filepath.Join(uploadDir, safeFileName)

		if err := c.SaveFile(fileHeader, savePath); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"success": false,
				"message": "Could not save file locally",
				"error":   err.Error(),
			})
		}

		finalUrl = fmt.Sprintf("/uploads/store_%s/%s", store.Slug, safeFileName)
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success": true,
		"message": "Image uploaded successfully",
		"url":     finalUrl,
	})
}
