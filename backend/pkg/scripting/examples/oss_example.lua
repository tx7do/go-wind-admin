-- Example: Using the OSS (Object Storage Service) API
-- This script demonstrates various OSS operations

local log = require "kratos_logger"
local oss = require "kratos_oss"
local hook = require "kratos_hook"

log.info("========================================")
log.info("  OSS API Example")
log.info("========================================")

-- Example 1: Get bucket name for content type
-- This helps you determine which bucket to use based on file type
local function example_get_bucket()
    log.info("\n--- Example 1: Get Bucket for Content Type ---")

    local jpeg_bucket = oss.get_bucket_for_type("image/jpeg")
    log.info("JPEG images → " .. jpeg_bucket)

    local png_bucket = oss.get_bucket_for_type("image/png")
    log.info("PNG images → " .. png_bucket)

    local mp4_bucket = oss.get_bucket_for_type("video/mp4")
    log.info("MP4 videos → " .. mp4_bucket)

    local pdf_bucket = oss.get_bucket_for_type("application/pdf")
    log.info("PDF documents → " .. pdf_bucket)
end

-- Example 2: Get presigned upload URL
-- This allows clients to upload files directly to OSS
local function example_upload_url()
    log.info("\n--- Example 2: Get Presigned Upload URL ---")

    local result = oss.upload_url({
        content_type = "image/jpeg",
        file_name = "avatar.jpg",
        file_path = "users/123"
    })

    log.info("✓ Upload URL generated:")
    log.info("  Upload URL: " .. result.upload_url)
    log.info("  Download URL: " .. result.download_url)
    log.info("  Object name: " .. result.object_name)
    log.info("  Bucket: " .. result.bucket_name)

    return result
end

-- Example 3: List files in a bucket
local function example_list_files()
    log.info("\n--- Example 3: List Files in Bucket ---")

    local files = oss.list_files({
        bucket_name = "images",
        folder = "users/",
        recursive = true
    })

    log.info("✓ Found " .. #files .. " files in 'images' bucket")

    for i, file in ipairs(files) do
        log.info(string.format(
            "  [%d] %s (%d bytes, modified: %s)",
            i,
            file.key,
            file.size,
            os.date("%Y-%m-%d %H:%M:%S", file.last_modified)
        ))
    end
end

-- Example 4: Upload file content directly
local function example_upload_file()
    log.info("\n--- Example 4: Direct File Upload ---")

    -- In a real scenario, you would read file content from somewhere
    local file_content = "This is test file content"

    local ok, url_or_error = oss.upload_file(
        "documents",
        "test/sample.txt",
        file_content
    )

    if ok then
        log.info("✓ File uploaded successfully")
        log.info("  Download URL: " .. url_or_error)
        return url_or_error
    else
        log.error("✗ Upload failed: " .. url_or_error)
        return nil
    end
end

-- Example 5: Delete a file
local function example_delete_file()
    log.info("\n--- Example 5: Delete File ---")

    local ok, error_msg = oss.delete_file("documents", "test/sample.txt")

    if ok then
        log.info("✓ File deleted successfully")
    else
        log.error("✗ Delete failed: " .. error_msg)
    end
end

-- Example 6: Ensure bucket exists
local function example_ensure_bucket()
    log.info("\n--- Example 6: Ensure Bucket Exists ---")

    local ok, error_msg = oss.ensure_bucket("custom-bucket")

    if ok then
        log.info("✓ Bucket 'custom-bucket' exists or was created")
    else
        log.error("✗ Bucket creation failed: " .. error_msg)
    end
end

-- Example 7: Hook integration - Auto-upload user avatars
hook.register("user.avatar_uploaded", "Process user avatar upload", function(ctx)
    log.info("\n--- Hook: Processing Avatar Upload ---")

    local user_id = ctx.get("user_id")
    local content_type = ctx.get("content_type")
    local file_data = ctx.get("file_data")

    -- Get presigned URL for upload
    local result = oss.upload_url({
        content_type = content_type,
        file_name = "avatar.jpg",
        file_path = "users/" .. user_id .. "/avatars"
    })

    log.info("✓ Avatar upload URL generated for user " .. user_id)

    -- Store the URLs in context for the application to use
    ctx.set("upload_url", result.upload_url)
    ctx.set("download_url", result.download_url)
    ctx.set("object_name", result.object_name)

    return true
end)

-- Example 8: Hook integration - Clean up old files
hook.register("user.deleted", "Clean up user files", function(ctx)
    log.info("\n--- Hook: Cleaning Up User Files ---")

    local user_id = ctx.get("user_id")

    -- List all files for this user
    local files = oss.list_files({
        bucket_name = "images",
        folder = "users/" .. user_id .. "/",
        recursive = true
    })

    log.info("Found " .. #files .. " files to delete")

    -- Delete each file
    local deleted_count = 0
    for i, file in ipairs(files) do
        local ok = oss.delete_file("images", file.key)
        if ok then
            deleted_count = deleted_count + 1
        end
    end

    log.info("✓ Deleted " .. deleted_count .. "/" .. #files .. " files")

    ctx.set("deleted_files", deleted_count)
    return true
end)

-- Run examples (commented out - uncomment to test specific examples)
-- example_get_bucket()
-- example_upload_url()
-- example_list_files()
-- example_upload_file()
-- example_delete_file()
-- example_ensure_bucket()

log.info("\n========================================")
log.info("  OSS hooks registered successfully")
log.info("========================================")
