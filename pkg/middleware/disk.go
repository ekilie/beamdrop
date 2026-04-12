package middleware

import (
	"net/http"

	"github.com/ekilie/beamdrop/pkg/errors"
	"github.com/ekilie/beamdrop/pkg/storage"
)

// MaxStorageCheck rejects write requests when usage exceeds maxBytes.
// If maxBytes is 0, the check is disabled (unlimited storage).
func MaxStorageCheck(sharedDir string, maxBytes int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if maxBytes <= 0 {
				next.ServeHTTP(w, r)
				return
			}

			// Only check on write methods
			switch r.Method {
			case http.MethodPost, http.MethodPut, http.MethodPatch:
			default:
				next.ServeHTTP(w, r)
				return
			}

			usage, err := storage.GetDirStorageUsage(sharedDir)
			if err != nil {
				errors.InternalError("Failed to check storage usage").WithCause(err).WriteHTTPResponse(w)
				return
			}

			if usage.UsedBytes >= uint64(maxBytes) {
				errors.StorageFull().WriteHTTPResponse(w)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
