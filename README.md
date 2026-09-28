# Clipit

Clipit is a small, self-hosted image library with resumable uploads. A Go server serves the web UI and API; uploaded images and their metadata are kept on local disk. There are no paid APIs, managed database subscriptions, cloud credentials, or runtime CDN dependencies.

## Features

- Responsive image gallery with search, download, clipboard copy, and delete.
- Copy the image itself to the clipboard as PNG and paste it into compatible chat inputs.
- Drag-and-drop and file-picker uploads for PNG, JPEG, and GIF.
- Resumable 512 KiB chunks with automatic retry and browser-side session recovery.
- Server-side image format and dimension validation (maximum 2 MiB and 16 megapixels).
- Metadata and image files persist across server restarts.
- Tailwind CSS is built locally and embedded in the Go executable.
- Health endpoint at `/health`.

## Run locally

Requirements: Go 1.26.1 or newer, Node.js 20 or newer, and npm.

```sh
npm install
npm start
```

Open <http://localhost:8080>. `npm start` compiles Tailwind and starts the Go server. Set `PORT` to change the listen port and `DATA_DIR` to change the storage directory. Defaults are `8080` and `./data`.

For a standalone Go binary:

```sh
npm install
npm run build
./bin/clipit
```

The built binary embeds the frontend assets. Keep the `data/` directory backed up: it contains both uploaded originals and the JSON metadata index.

## API

All routes are served by the same Go process and use JSON unless noted.

| Method | Route | Purpose |
| --- | --- | --- |
| `GET` | `/health` | Liveness check |
| `GET` | `/api/media` | List completed images |
| `POST` | `/api/uploads` | Create a resumable upload session with `{ "name": "photo.png", "size": 123 }` |
| `GET` | `/api/uploads/{id}` | Read upload state and completed chunk numbers |
| `PUT` | `/api/uploads/{id}/parts/{part}` | Send a raw binary chunk |
| `POST` | `/api/uploads/{id}/complete` | Validate and finalize an image |
| `GET` | `/api/media/{id}/file` | Serve the validated image |
| `DELETE` | `/api/media/{id}` | Delete an image and metadata |

Chunk numbers start at zero. A chunk must be exactly 512 KiB except the final chunk. Repeating an identical chunk is safe; conflicting chunk data is rejected.

## Project layout

```text
cmd/api/                 Go server and embedded web assets
cmd/api/web/             HTML, browser JavaScript, generated Tailwind CSS
frontend/input.css       Tailwind entry stylesheet
internal/upload/         Upload API, validation, file store, tests
package.json             Frontend asset build and local run scripts
tailwind.config.js       Tailwind theme and content paths
```

## Deployment and security boundaries

The MVP is intentionally self-hosted and single-instance. It runs locally at no cost and can be deployed to infrastructure you already control without changing the application architecture. It does not depend on any free cloud tier whose availability, limits, or billing terms may change. A public host must provide persistent writable storage; ephemeral container filesystems will lose uploads on restart or redeploy.

This MVP has no authentication or per-user ownership, rate limiting, quotas, malware scanning, background worker, database, CDN, or multi-instance coordination. Anyone who can reach the service can list and delete its images. Do not expose it to the public internet with private images until authentication and authorization are added. Files are stored locally, so multiple server instances cannot safely share the catalog. Back up `DATA_DIR` before upgrading or relocating the service.

The Go API uses only the Go standard library and `github.com/google/uuid`; the browser interface uses no runtime third-party services. Tailwind CSS is an open-source build dependency installed by npm.
