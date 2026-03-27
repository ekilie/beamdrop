# Failure Scenarios

This document describes how Beamdrop behaves under various failure conditions, the recovery procedures for each, and the data integrity guarantees the system provides.

## Table of Contents

1. [Crash During Upload](#crash-during-upload)
2. [Crash During Delete](#crash-during-delete)
3. [Disk Full](#disk-full)
4. [Database Unavailable](#database-unavailable)
5. [Recovery Procedures](#recovery-procedures)
6. [Data Integrity Guarantees](#data-integrity-guarantees)

---

## Crash During Upload

### Expected Behavior

Beamdrop uses an **atomic write** strategy for all uploads (see `pkg/storage/atomic.go`). The sequence is:

1. Data is written to a temporary file (prefixed `.beamdrop_tmp_`) in the same directory as the target.
2. The temporary file is flushed to disk with `fsync`.
3. The temporary file is atomically renamed to the final path (POSIX `rename(2)`).
4. The parent directory metadata is synced.

**If the server crashes at any point during this process:**

| Crash timing                        | Effect on filesystem                                                                                                     | Effect on database                                                                |
| ----------------------------------- | ------------------------------------------------------------------------------------------------------------------------ | --------------------------------------------------------------------------------- |
| During write to temp file           | Orphaned `.beamdrop_tmp_*` file remains on disk. The target file is untouched.                                           | No database record is created (uploads update stats only after successful write). |
| After `fsync`, before rename        | Complete temp file on disk, but the target file does not exist yet.                                                      | No database record is created.                                                    |
| After rename, before directory sync | The file is in place. Directory metadata may not be persisted, but this is a cosmetic risk the file data itself is safe. | Stats update may be lost.                                                         |

**Client behavior:** The upload HTTP request will fail (connection reset). The frontend shows an error state and allows the user to retry.

### Why Files Are Never Corrupted

Because `AtomicWriteFile` writes to a temporary file first and only renames after a successful `fsync`, a crash can never leave a **partially written target file** on disk. The rename operation is atomic on POSIX systems the file either appears in full or not at all.

---

## Crash During Delete

### Expected Behavior

File deletion (trash) uses `os.Rename` to move the file from its source path into `.beamdrop_trash/` (see `beam/server/handlers/file_operations.go`). Rename is atomic on POSIX filesystems.

**If the server crashes:**

| Crash timing                 | Effect                                                                          |
| ---------------------------- | ------------------------------------------------------------------------------- |
| Before `os.Rename` completes | The file remains in its original location. No data is lost.                     |
| After `os.Rename` completes  | The file is in the trash directory. The HTTP response may not reach the client. |

**If the client receives a connection reset:** The file may or may not have been trashed. The user can check the file list if the file is still present, the delete did not complete and can be retried safely.

### Associated Database Records

Database records that reference deleted files (starred files, shareable links) are cleaned up by the `OrphanCleaner` background job, which runs once at startup and then every hour (see `pkg/db/cleanup.go`). Even if a crash prevents immediate cleanup, these orphaned records are removed automatically on the next run.

---

## Disk Full

### Expected Behavior

When the disk is full, file write operations fail at the OS level. Beamdrop handles this as follows:

| Operation             | Behavior                                                                                                                                            |
| --------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Upload (web/REST)** | `AtomicWriteFile` fails during temp file write. The temp file is removed via `Abort()`. The server returns HTTP 500 with error code `WRITE_FAILED`. |
| **Upload (S3 API)**   | Object write fails. Returns S3-compatible error response with `WRITE_FAILED` code.                                                                  |
| **Copy/Move**         | File operation fails. Returns HTTP 500 with `IO_ERROR`.                                                                                             |
| **Database writes**   | SQLite writes fail. The transaction is rolled back automatically.                                                                                   |

**Error response format** (structured errors from `pkg/errors/errors.go`):

```json
{
  "error": {
    "code": "WRITE_FAILED",
    "message": "Failed to save file",
    "category": "STORAGE",
    "retryable": false
  }
}
```

**Client behavior:** The frontend displays an error toast. The user should free disk space and retry. Since all writes are atomic, no partial files are left behind.

### Storage Full Detection

Beamdrop defines the `STORAGE_FULL` and `QUOTA_EXCEEDED` error codes. When a write fails, the OS error is wrapped and surfaced through the structured error system. Administrators can monitor disk space via standard OS tools or Grafana dashboards (see `docs/grafana-dashboard.json`).

---

## Database Unavailable

### Expected Behavior

Beamdrop uses SQLite (`pkg/db/db.go`), which stores all metadata in a single file (`~/.beamdrop/beamdrop.db` by default). The database can become unavailable if:

- The database file is deleted or corrupted
- The disk containing the database is full
- File-level locks are held by another process

**Impact by feature area:**

| Feature                  | Impact when DB is unavailable                                                                              |
| ------------------------ | ---------------------------------------------------------------------------------------------------------- |
| **File upload/download** | Uploads and downloads still work (they operate on the filesystem). Upload statistics will not be recorded. |
| **Starred files**        | Starring/unstarring fails with `DATABASE_ERROR`.                                                           |
| **Shareable links**      | Creating/resolving links fails with `DATABASE_ERROR`.                                                      |
| **Presigned URLs**       | Creating/validating presigned URLs fails.                                                                  |
| **API keys**             | Authentication via API keys fails, blocking protected API access.                                          |
| **Metrics/stats**        | Stats updates fail silently; the server continues to operate.                                              |

**Error response format:**

```json
{
  "error": {
    "code": "DATABASE_ERROR",
    "message": "Failed to star file",
    "category": "INTERNAL",
    "retryable": true
  }
}
```

### Transaction Safety

All multi-step database operations use `WithTransaction` (`pkg/db/transaction.go`), which:

1. Begins a transaction.
2. Executes the operation.
3. Rolls back automatically if the operation returns an error **or** panics.
4. Commits only on success.

This ensures the database is never left in a partially updated state.

### Cross-System Transactions (Saga Pattern)

Operations that span both the database and the filesystem (e.g., creating a shareable link that references a file) use the saga pattern (`pkg/db/saga.go`). Each step has a compensating action:

1. Steps execute in order.
2. If any step fails, all previously completed steps are rolled back in reverse order.
3. Compensation errors are logged but do not mask the original failure.

---

## Recovery Procedures

### After a Server Crash

1. **Restart the server.** On startup, Beamdrop automatically:
   - Calls `CleanupOrphanedTempFiles` (`pkg/storage/atomic.go`) to remove any `.beamdrop_tmp_*` files left by interrupted uploads.
   - Starts the `OrphanCleaner` (`pkg/db/cleanup.go`), which immediately runs a cleanup pass to remove database records pointing to files that no longer exist.

2. **Verify file integrity.** All successfully uploaded files are intact the atomic write strategy guarantees this. Files that were mid-upload at crash time will not exist (only their temp files did, and those are cleaned up).

3. **Check logs.** Review server logs for any `slog.Error` entries from the cleanup pass, which indicate files that could not be cleaned up (e.g., permission issues).

### After Disk Full

1. **Free disk space.** Remove unnecessary files, empty the trash (`.beamdrop_trash/` in the shared directory), or expand the volume.
2. **Restart the server** if it exited due to the condition. Startup cleanup will remove any orphaned temp files.
3. **Retry failed operations.** No manual data repair is needed atomic writes ensure clean state.

### After Database Corruption

1. **Stop the server.**
2. **Restore from backup** if available. The database is a single SQLite file at the configured path (default `~/.beamdrop/beamdrop.db`).
3. **If no backup exists:** Delete the database file and restart the server. Beamdrop will create a fresh database. **Data loss:** starred files, shareable links, presigned URLs, API keys, and upload statistics will be lost. Uploaded files on the filesystem are unaffected.
4. **Re-create metadata** as needed (re-star files, re-create shareable links).

### After Database Lock Contention

SQLite uses file-level locking. If another process holds a lock:

1. Identify and stop the other process accessing the database file.
2. Restart Beamdrop. Normal operation will resume.

---

## Data Integrity Guarantees

### File Data

| Guarantee                    | Mechanism                                                                                            |
| ---------------------------- | ---------------------------------------------------------------------------------------------------- |
| **No partial files**         | Atomic write (temp file → `fsync` → `rename`).                                                       |
| **No silent corruption**     | `fsync` flushes data to durable storage before rename. S3 API uploads support MD5 ETag verification. |
| **Crash-safe uploads**       | Incomplete uploads leave only temp files, which are cleaned up on restart.                           |
| **Crash-safe deletes**       | `os.Rename` is atomic the file is either in the original location or in trash, never lost.           |
| **Concurrent access safety** | The S3 API layer uses per-object read/write locks (`pkg/storage/locks.go`) with 30-second timeouts.  |

### Database (Metadata)

| Guarantee                            | Mechanism                                                                                                           |
| ------------------------------------ | ------------------------------------------------------------------------------------------------------------------- |
| **ACID transactions**                | SQLite provides full ACID compliance. All multi-step operations use `WithTransaction`.                              |
| **No partial updates**               | Transactions roll back automatically on error or panic.                                                             |
| **Cross-system consistency**         | Saga pattern rolls back filesystem changes if DB operations fail (and vice versa).                                  |
| **Eventual consistency for orphans** | `OrphanCleaner` runs hourly to remove records referencing deleted files, expired links, and expired presigned URLs. |

### Known Limitations

| Limitation                                      | Detail                                                                                                                                                                                        |
| ----------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Trash does not preserve directory structure** | Only the filename is preserved when moving to `.beamdrop_trash/`. If two files with the same name from different directories are trashed, the second overwrites the first.                    |
| **No trash restoration API**                    | Recovery from trash requires manual filesystem operations.                                                                                                                                    |
| **No trash expiration policy**                  | Trashed files persist until manually removed.                                                                                                                                                 |
| **In-memory locks are lost on restart**         | Object locks (`pkg/storage/locks.go`) exist only in memory. A crash releases all locks, which is safe because operations either completed atomically or will be cleaned up on restart.        |
| **Single-file SQLite database**                 | Not suitable for high-concurrency deployments. For heavy workloads, use an external reverse proxy for connection management and ensure only one Beamdrop instance accesses the database file. |
| **No built-in backup**                          | Administrators should set up external backups for the SQLite database file. The file can be safely copied while the server is running (SQLite supports this via its WAL mode).                |
