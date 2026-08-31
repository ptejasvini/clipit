# Clipit

Clipit is a resumable media upload platform built with Go.

It allows authenticated users to upload images up to 2 MB using
chunk-based multipart uploads. Uploaded media is validated, processed
asynchronously, and made available for fast delivery.

The project is designed to demonstrate real-world backend engineering
concepts such as:

- Resumable uploads
- Chunk-based multipart uploads
- Authentication and authorization
- Image validation
- Distributed state management
- Asynchronous processing
- Event-driven architecture
- Object storage
- CDN-based media delivery
- Concurrency and idempotency
- Resource limits and cleanup

---

## Problem Statement

Traditional file uploads have a simple failure mode:

A user uploads a file, the connection fails near the end, and the entire
upload has to start again.

CLIPiT solves this by breaking an image into smaller chunks.

If a chunk fails, the client can retry only that chunk instead of
re-uploading the entire file.

The system also separates the upload path from media processing so that
image processing does not block the API request.

---

## Core Features

### Authentication

- User registration
- User login
- JWT-based authentication
- Protected media resources
- User-level media ownership

### Resumable Uploads

- Maximum image size: 2 MB
- Chunk-based upload
- Multipart HTTP requests
- Upload session tracking
- Chunk validation
- Resume interrupted uploads
- Retry failed chunks
- Idempotent chunk uploads
- Upload expiration

### Image Validation

- Validate file size
- Validate MIME type
- Validate image signatures/magic bytes
- Validate image dimensions
- Decode image before accepting it
- Reject malformed or unsupported files

### Asynchronous Processing

After an upload is completed:

1. The API validates the upload.
2. A media event is published.
3. A background worker consumes the event.
4. The worker processes the image.
5. Optimized variants are generated.
6. Media metadata is updated.
7. The processed image becomes available.

### Media Processing

The processing pipeline can generate:

- Original/normalized image
- Optimized image
- WebP version
- Thumbnail

### Copy Image

The web application provides a **Copy Image** action using the browser
Clipboard API.

The user can copy the image and paste it into supported applications
without manually downloading it first.

---

## Architecture

```text
                         ┌─────────────────┐
                         │     Browser     │
                         │                 │
                         │ React + TS      │
                         └────────┬────────┘
                                  │
                                  │ HTTPS
                                  ▼
                         ┌─────────────────┐
                         │    Go API       │
                         │                 │
                         │ Auth            │
                         │ Upload API      │
                         │ Media API       │
                         └────────┬────────┘
                                  │
               ┌──────────────────┼──────────────────┐
               │                  │                  │
               ▼                  ▼                  ▼
        ┌─────────────┐    ┌─────────────┐    ┌─────────────┐
        │ PostgreSQL  │    │    Redis    │    │   Storage   │
        │             │    │             │    │             │
        │ Metadata    │    │ Upload      │    │ Temporary   │
        │ Users       │    │ State       │    │ Media       │
        │ Media       │    │ TTL         │    │ Objects     │
        └─────────────┘    └─────────────┘    └──────┬──────┘
                                                     │
                                                     │
                                  Upload Completed   │
                                         │           │
                                         ▼           │
                                  ┌─────────────┐    │
                                  │    Kafka    │    │
                                  │             │    │
                                  │ Media Events│    │
                                  └──────┬──────┘    │
                                         │           │
                                         ▼           │
                                  ┌─────────────┐    │
                                  │ Go Worker   │◄───┘
                                  │             │
                                  │ Validation  │
                                  │ Processing  │
                                  │ Optimization│
                                  └──────┬──────┘
                                         │
                                         ▼
                                  ┌─────────────┐
                                  │   Object    │
                                  │   Storage   │
                                  └──────┬──────┘
                                         │
                                         ▼
                                  ┌─────────────┐
                                  │     CDN     │
                                  └──────┬──────┘
                                         │
                                         ▼
                                     Browser
