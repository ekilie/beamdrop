package main

import (
	"context"
	"fmt"
	"log"
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

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	config := loadConfig()

	beamdrop, err := client.New(client.Config{
		BaseURL:     config.BaseURL,
		AccessKeyID: config.AccessKeyID,
		SecretKey:   config.SecretKey,
	})
	if err != nil {
		log.Fatal(err)
	}

	lifecycleBucket := config.Bucket + "-lifecycle"

	// 1. Create a fresh bucket (will fail if it already exists).
	logStep("creating bucket (strict)")
	result, err := beamdrop.CreateBucket(ctx, lifecycleBucket)
	if err != nil {
		apiErr, ok := err.(*client.APIError)
		if ok && apiErr.StatusCode == 409 {
			log.Printf("bucket %q already exists (409) — this is expected on re-run\n", lifecycleBucket)
		} else {
			log.Fatal(err)
		}
	} else {
		log.Printf("bucket created: %s at %s\n", result.Bucket, result.Location)
	}

	// 2. Idempotent creation — safe to call repeatedly.
	logStep("creating bucket (idempotent)")
	result, err = beamdrop.CreateBucketIfNotExists(ctx, lifecycleBucket)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("bucket %q exists=%t\n", result.Bucket, result.Exists)

	// 3. List all buckets and show the lifecycle bucket.
	logStep("listing buckets")
	buckets, err := beamdrop.ListBuckets(ctx)
	if err != nil {
		log.Fatal(err)
	}
	for _, b := range buckets.Buckets {
		status := " "
		if b.Name == lifecycleBucket {
			status = ">"
		}
		log.Printf("%s bucket: %s created=%s\n", status, b.Name, b.CreatedAt.Format(time.RFC3339))
	}

	// 4. Check if the bucket exists (HEAD).
	logStep("checking bucket existence")
	exists, err := beamdrop.BucketExists(ctx, lifecycleBucket)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("bucket %q exists: %t\n", lifecycleBucket, exists)

	// 5. Upload a dummy object so deletion of a non-empty bucket fails.
	logStep("uploading object to non-empty bucket")
	_, err = beamdrop.PutObject(ctx, lifecycleBucket, "keep.txt", []byte("delete me first"))
	if err != nil {
		log.Fatal(err)
	}
	log.Println("uploaded keep.txt")

	// 6. Try to delete the non-empty bucket (should get 409).
	logStep("deleting non-empty bucket (expect 409)")
	err = beamdrop.DeleteBucket(ctx, lifecycleBucket)
	if err != nil {
		apiErr, ok := err.(*client.APIError)
		if ok && apiErr.StatusCode == 409 {
			log.Printf("non-empty bucket rejected (409): %s\n", apiErr.Message)
		} else {
			log.Fatal(err)
		}
	} else {
		log.Println("WARNING: non-empty bucket was deleted (unexpected)")
	}

	// 7. Delete the object, then delete the empty bucket.
	logStep("deleting object and clearing bucket")
	if err := beamdrop.DeleteObject(ctx, lifecycleBucket, "keep.txt"); err != nil {
		log.Fatal(err)
	}
	log.Println("deleted keep.txt")

	if err := beamdrop.DeleteBucket(ctx, lifecycleBucket); err != nil {
		log.Fatal(err)
	}
	log.Printf("deleted bucket %q\n", lifecycleBucket)

	// 8. Verify deletion.
	logStep("verifying deletion")
	exists, err = beamdrop.BucketExists(ctx, lifecycleBucket)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("bucket %q exists after deletion: %t\n", lifecycleBucket, exists)

	fmt.Println("bucket lifecycle example completed successfully")
}

type exampleConfig struct {
	BaseURL     string
	AccessKeyID string
	SecretKey   string
	Bucket      string
}

func loadConfig() exampleConfig {
	return exampleConfig{
		BaseURL:     requiredEnv("BEAMDROP_BASE_URL"),
		AccessKeyID: requiredEnv("BEAMDROP_ACCESS_KEY_ID"),
		SecretKey:   requiredEnv("BEAMDROP_SECRET_KEY"),
		Bucket:      optionalEnv("BEAMDROP_BUCKET", "beamdrop-go-example"),
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
