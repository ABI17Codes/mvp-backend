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

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
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

	// R2 configuration
	accountID := os.Getenv("R2_ACCOUNT_ID")
	accessKeyId := os.Getenv("R2_ACCESS_KEY_ID")
	accessKeySecret := os.Getenv("R2_SECRET_ACCESS_KEY")
	bucketName := os.Getenv("R2_BUCKET_NAME")
	endpoint := os.Getenv("R2_ENDPOINT")

	if accountID == "" || accessKeyId == "" || accessKeySecret == "" || bucketName == "" || endpoint == "" {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "R2 configuration is incomplete",
		})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion("auto"),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKeyId, accessKeySecret, "")),
	)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Failed to load R2 configuration",
			"error":   err.Error(),
		})
	}

	// Create an S3 client
	s3Client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(endpoint)
	})

	objectKey := fmt.Sprintf("kadaitheru/%s/%s_%s%s", store.Slug, uuid.New().String(), strings.TrimSuffix(filepath.Base(fileHeader.Filename), filepath.Ext(fileHeader.Filename)), filepath.Ext(fileHeader.Filename))

	file, err := fileHeader.Open()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Could not read uploaded file",
			"error":   err.Error(),
		})
	}
	defer file.Close()

	_, uploadErr := s3Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(bucketName),
		Key:         aws.String(objectKey),
		Body:        file,
		ContentType: aws.String(mime),
	})

	if uploadErr != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "R2 upload failed",
			"error":   uploadErr.Error(),
		})
	}

	publicDomain := os.Getenv("R2_PUBLIC_DOMAIN")
	var publicURL string
	if publicDomain != "" {
		publicURL = fmt.Sprintf("%s/%s", strings.TrimRight(publicDomain, "/"), objectKey)
	} else {
		// Fallback to S3 endpoint URL (might not be publicly accessible by default on R2)
		publicURL = fmt.Sprintf("%s/%s/%s", strings.TrimRight(endpoint, "/"), bucketName, objectKey)
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success": true,
		"message": "Image uploaded successfully",
		"url":     publicURL,
	})
}

// PublicUploadImage handles image uploads for customers without JWT (e.g. payment screenshots).
func PublicUploadImage(c fiber.Ctx) error {
	fileHeader, err := c.FormFile("image")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Image file is required",
		})
	}

	ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
	if ext != ".jpg" && ext != ".jpeg" && ext != ".png" && ext != ".webp" && ext != ".svg" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid file format",
		})
	}

	// 2 MB max size
	if fileHeader.Size > 2*1024*1024 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "File size exceeds 2MB limit",
		})
	}

	endpoint := os.Getenv("R2_ENDPOINT")
	if endpoint == "" {
		uploadDir := "./uploads"
		if err := os.MkdirAll(uploadDir, 0755); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"success": false,
				"message": "Could not create upload directory",
			})
		}
		newFilename := fmt.Sprintf("%d-%s", time.Now().UnixNano(), fileHeader.Filename)
		filePath := filepath.Join(uploadDir, newFilename)

		if err := c.SaveFile(fileHeader, filePath); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"success": false,
				"message": "Could not save file locally",
			})
		}

		publicURL := fmt.Sprintf("/uploads/%s", newFilename)
		return c.Status(fiber.StatusCreated).JSON(fiber.Map{
			"success": true,
			"message": "Image uploaded locally",
			"url":     publicURL,
		})
	}

	bucketName := os.Getenv("R2_BUCKET")
	accessKeyID := os.Getenv("R2_ACCESS_KEY_ID")
	secretAccessKey := os.Getenv("R2_SECRET_ACCESS_KEY")

	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKeyID, secretAccessKey, "")),
		config.WithRegion("auto"),
	)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Could not load R2 configuration",
		})
	}

	s3Client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(endpoint)
	})

	mime := "application/octet-stream"
	switch ext {
	case ".jpg", ".jpeg":
		mime = "image/jpeg"
	case ".png":
		mime = "image/png"
	case ".webp":
		mime = "image/webp"
	case ".svg":
		mime = "image/svg+xml"
	}

	newFilename := fmt.Sprintf("%d-%s", time.Now().UnixNano(), uuid.New().String()+ext)
	objectKey := fmt.Sprintf("store_%s/%s", "public", newFilename)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	file, err := fileHeader.Open()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Could not read uploaded file",
		})
	}
	defer file.Close()

	_, uploadErr := s3Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(bucketName),
		Key:         aws.String(objectKey),
		Body:        file,
		ContentType: aws.String(mime),
	})

	if uploadErr != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "R2 upload failed",
		})
	}

	publicDomain := os.Getenv("R2_PUBLIC_DOMAIN")
	var publicURL string
	if publicDomain != "" {
		publicURL = fmt.Sprintf("%s/%s", strings.TrimRight(publicDomain, "/"), objectKey)
	} else {
		publicURL = fmt.Sprintf("%s/%s/%s", strings.TrimRight(endpoint, "/"), bucketName, objectKey)
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success": true,
		"message": "Image uploaded successfully",
		"url":     publicURL,
	})
}

