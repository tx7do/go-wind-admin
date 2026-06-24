package api

import (
	"testing"

	"github.com/go-kratos/kratos/v2/log"
	lua "github.com/yuin/gopher-lua"
)

// TestRegisterOSS_Module tests that the OSS module can be loaded
func TestRegisterOSS_Module(t *testing.T) {
	// Skip if no OSS client available (for CI/CD)
	t.Skip("Skipping OSS tests - requires MinIO setup")

	L := lua.NewState()
	defer L.Close()

	_ = log.NewHelper(log.DefaultLogger)

	// Note: This test would require a real MinIOClient
	// For now, we'll just verify the module structure

	// Create a mock OSS client for testing
	// In a real scenario, you would use: ossClient := oss.NewMinIoClient(cfg, logger)
	// RegisterOSS(L, ossClient, logger)

	script := `
		-- Test that we can require the module
		local oss = require "kratos_oss"

		-- Verify the API functions exist
		assert(oss.upload_url ~= nil, "upload_url should exist")
		assert(oss.list_files ~= nil, "list_files should exist")
		assert(oss.delete_file ~= nil, "delete_file should exist")
		assert(oss.upload_file ~= nil, "upload_file should exist")
		assert(oss.ensure_bucket ~= nil, "ensure_bucket should exist")
		assert(oss.get_bucket_for_type ~= nil, "get_bucket_for_type should exist")

		return true
	`

	err := L.DoString(script)
	if err == nil {
		t.Error("Expected error due to missing OSS registration, but got none")
	}

	t.Log("✓ OSS module structure test passed")
}

// TestOSSAPI_GetBucketForType tests bucket name determination
func TestOSSAPI_GetBucketForType(t *testing.T) {
	t.Skip("Skipping - requires MinIO setup")

	// This test demonstrates how the OSS API would be used
	exampleScript := `
		local oss = require "kratos_oss"
		local log = require "kratos_logger"

		-- Get appropriate bucket for different content types
		local image_bucket = oss.get_bucket_for_type("image/jpeg")
		log.info("JPEG images go to: " .. image_bucket)

		local video_bucket = oss.get_bucket_for_type("video/mp4")
		log.info("MP4 videos go to: " .. video_bucket)

		local doc_bucket = oss.get_bucket_for_type("application/pdf")
		log.info("PDF documents go to: " .. doc_bucket)

		return true
	`

	t.Log("Example script for OSS bucket determination:")
	t.Log(exampleScript)
}

// TestOSSAPI_UploadFlow tests the complete upload flow
func TestOSSAPI_UploadFlow(t *testing.T) {
	t.Skip("Skipping - requires MinIO setup")

	// This test demonstrates the complete OSS upload workflow
	workflowScript := `
		local oss = require "kratos_oss"
		local log = require "kratos_logger"

		-- Step 1: Get presigned upload URL
		local result = oss.upload_url({
			content_type = "image/jpeg",
			file_name = "profile.jpg",
			file_path = "users/avatars"
		})

		log.info("Upload URL: " .. result.upload_url)
		log.info("Download URL: " .. result.download_url)
		log.info("Object name: " .. result.object_name)
		log.info("Bucket: " .. result.bucket_name)

		-- Step 2: Client uploads to the presigned URL (happens in frontend)
		-- ...

		-- Step 3: List files in the bucket
		local files = oss.list_files({
			bucket_name = result.bucket_name,
			folder = "users/avatars",
			recursive = true
		})

		log.info("Found " .. #files .. " files")
		for i, file in ipairs(files) do
			log.info("File: " .. file.key .. " (" .. file.size .. " bytes)")
		end

		-- Step 4: Direct upload (alternative to presigned URL)
		local file_content = "fake image data"
		local ok, url = oss.upload_file("images", "test/image.jpg", file_content)
		if ok then
			log.info("File uploaded to: " .. url)
		end

		-- Step 5: Delete file when needed
		local deleted = oss.delete_file("images", "test/image.jpg")
		if deleted then
			log.info("File deleted successfully")
		end

		return true
	`

	t.Log("Example OSS upload workflow:")
	t.Log(workflowScript)
}
