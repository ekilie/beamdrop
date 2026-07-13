package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/ekilie/beamdrop/pkg/client"
	"github.com/joho/godotenv"
)

func main() {
	log.SetFlags(0)
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	config := loadConfig()

	// Authenticated client for creating objects and presigned URLs.
	beamdrop, err := client.New(client.Config{
		BaseURL:     config.BaseURL,
		AccessKeyID: config.AccessKeyID,
		SecretKey:   config.SecretKey,
	})
	if err != nil {
		log.Fatal(err)
	}

	downloader := &http.Client{Timeout: 30 * time.Second}

	// 1. Upload a private object.
	logStep("uploading private object")
	_, err = beamdrop.PutObject(ctx, config.Bucket, "private/secret-note.txt", []byte("this is a private file\n"))
	if err != nil {
		log.Fatal(err)
	}
	log.Println("uploaded private/secret-note.txt")

	// 2. Create a server-side presigned URL with a download limit.
	logStep("creating presigned URL (max 3 downloads)")
	expiresIn := int64(3600)
	maxDownloads := 3
	presigned, err := beamdrop.CreatePresignedURL(ctx, client.CreatePresignedURLRequest{
		Bucket:       config.Bucket,
		Key:          "private/secret-note.txt",
		Method:       http.MethodGet,
		ExpiresIn:    &expiresIn,
		MaxDownloads: &maxDownloads,
	})
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("presigned URL: %s\n", presigned.URL)
	log.Printf("token: %s (max %d downloads, expires in %d s)\n", presigned.Token, maxDownloads, expiresIn)

	// 3. Download using the presigned URL with a plain HTTP client (no API key).
	logStep("downloading via presigned URL (attempt 1)")
	body := downloadPlainHTTP(downloader, presigned.URL)
	log.Printf("got: %q\n", strings.TrimSpace(string(body)))

	logStep("downloading via presigned URL (attempt 2)")
	body = downloadPlainHTTP(downloader, presigned.URL)
	log.Printf("got: %q\n", strings.TrimSpace(string(body)))

	// 4. Show that direct (non-presigned) download would fail without auth.
	logStep("direct download without auth (expect 401)")
	directURL := config.BaseURL + "/api/v1/buckets/" + config.Bucket + "/private/secret-note.txt"
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, directURL, nil)
	resp, err := downloader.Do(req)
	if err != nil {
		log.Printf("direct download failed (expected): %v\n", err)
	} else {
		resp.Body.Close()
		log.Printf("direct download returned %d\n", resp.StatusCode)
	}

	// 5. Cleanup: delete the presigned URL and object.
	if config.Cleanup {
		logStep("cleanup")
		if err := beamdrop.DeletePresignedURL(ctx, presigned.Token); err != nil {
			log.Fatal(err)
		}
		log.Println("deleted presigned URL")
		if err := beamdrop.DeleteObject(ctx, config.Bucket, "private/secret-note.txt"); err != nil {
			log.Fatal(err)
		}
		log.Println("deleted object")
	}

	fmt.Println("presigned download example completed successfully")
}

func downloadPlainHTTP(httpClient *http.Client, url string) []byte {
	resp, err := httpClient.Get(url)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Fatalf("download failed: HTTP %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	return body
}

type exampleConfig struct {
	BaseURL     string
	AccessKeyID string
	SecretKey   string
	Bucket      string
	Cleanup     bool
}

func loadConfig() exampleConfig {
	return exampleConfig{
		BaseURL:     requiredEnv("BEAMDROP_BASE_URL"),
		AccessKeyID: requiredEnv("BEAMDROP_ACCESS_KEY_ID"),
		SecretKey:   requiredEnv("BEAMDROP_SECRET_KEY"),
		Bucket:      optionalEnv("BEAMDROP_BUCKET", "beamdrop-go-example"),
		Cleanup:     strings.EqualFold(optionalEnv("BEAMDROP_CLEANUP", "false"), "true"),
	}
}

func requiredEnv(key string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		log.Fatalf("missing required environment variable %s", key)
	}
	return value
}

func optionalEnv(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func logStep(name string) {
	log.Printf("\n== %s ==\n", name)
}
