package main

import (
	"bytes"
	"context"
	"fmt"
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

	logStep("checking bucket")
	bucketExists, err := beamdrop.BucketExists(ctx, config.Bucket)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("bucket %q exists before setup: %t\n", config.Bucket, bucketExists)

	logStep("ensuring bucket exists")
	bucketResult, err := beamdrop.CreateBucketIfNotExists(ctx, config.Bucket)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("bucket result: %+v\n", *bucketResult)

	logStep("uploading objects")
	firstUpload, err := beamdrop.PutObject(ctx, config.Bucket, "demo/hello.txt", []byte("hello beamdrop from the Go client example\n"))
	if err != nil {
		log.Fatal(err)
	}
	secondUpload, err := beamdrop.PutObjectReader(ctx, config.Bucket, "demo/notes/streamed.txt", bytes.NewBufferString("this object was uploaded with PutObjectReader\n"))
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("uploaded: %s (%d bytes)\n", firstUpload.Key, firstUpload.Size)
	log.Printf("uploaded: %s (%d bytes)\n", secondUpload.Key, secondUpload.Size)

	logStep("listing buckets")
	buckets, err := beamdrop.ListBuckets(ctx)
	if err != nil {
		log.Fatal(err)
	}
	for _, bucket := range buckets.Buckets {
		log.Printf("bucket: %s created=%s\n", bucket.Name, bucket.CreatedAt.Format(time.RFC3339))
	}

	logStep("listing objects with prefix")
	objects, err := beamdrop.ListObjects(ctx, config.Bucket, client.ListObjectsOptions{Prefix: "demo/", MaxKeys: 100})
	if err != nil {
		log.Fatal(err)
	}
	for _, object := range objects.Contents {
		log.Printf("object: %s size=%d etag=%s\n", object.Key, object.Size, object.ETag)
	}

	logStep("reading object metadata")
	metadata, err := beamdrop.HeadObject(ctx, config.Bucket, "demo/hello.txt")
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("metadata: contentType=%s contentLength=%d etag=%s\n", metadata.ContentType, metadata.ContentLength, metadata.ETag)

	logStep("downloading object")
	object, err := beamdrop.GetObject(ctx, config.Bucket, "demo/hello.txt")
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("downloaded body: %q\n", strings.TrimSpace(string(object.Body)))

	logStep("checking object existence")
	objectExists, err := beamdrop.ObjectExists(ctx, config.Bucket, "demo/notes/streamed.txt")
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("object exists: %t\n", objectExists)

	logStep("creating client-side presigned URL")
	clientPresignedURL, err := beamdrop.PresignObjectURL(http.MethodGet, config.Bucket, "demo/hello.txt", time.Now().UTC().Add(15*time.Minute))
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("client-side presigned URL: %s\n", clientPresignedURL)

	logStep("creating server-side presigned URL")
	expiresIn := int64(900)
	serverPresignedURL, err := beamdrop.CreatePresignedURL(ctx, client.CreatePresignedURLRequest{
		Bucket:    config.Bucket,
		Key:       "demo/hello.txt",
		Method:    http.MethodGet,
		ExpiresIn: &expiresIn,
	})
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("server-side presigned URL: %s\n", serverPresignedURL.URL)

	logStep("listing server-side presigned URLs")
	presignedURLs, err := beamdrop.ListPresignedURLs(ctx)
	if err != nil {
		log.Fatal(err)
	}
	for _, presignedURL := range presignedURLs.URLs {
		log.Printf("presigned token=%s method=%s key=%s\n", presignedURL.Token, presignedURL.Method, presignedURL.Key)
	}

	logStep("fetching a server-side presigned URL by token")
	presignedURL, err := beamdrop.GetPresignedURL(ctx, serverPresignedURL.Token)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("retrieved presigned URL: token=%s bucket=%s key=%s\n", presignedURL.Token, presignedURL.Bucket, presignedURL.Key)

	if config.Cleanup {
		logStep("cleanup")
		if err := beamdrop.DeletePresignedURL(ctx, serverPresignedURL.Token); err != nil {
			log.Fatal(err)
		}
		if err := beamdrop.DeleteObject(ctx, config.Bucket, "demo/notes/streamed.txt"); err != nil {
			log.Fatal(err)
		}
		if err := beamdrop.DeleteObject(ctx, config.Bucket, "demo/hello.txt"); err != nil {
			log.Fatal(err)
		}
		log.Println("deleted demo objects and presigned URL")
	}

	fmt.Println("example completed successfully")
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
