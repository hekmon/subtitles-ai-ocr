package main

import (
	"context"
	"encoding/json"
	"fmt"
	"image"
	"strconv"

	"github.com/openai/openai-go"
)

const (
	batchMaxRequests = 50_000
	batchMaxSize     = 200_000_000 // 200 MB
)

func OCRBatched(ctx context.Context, imgSubs []PGSSubtitle, client openai.Client, model string, italic, debug bool) (txtSubs SRTSubtitles, err error) {
	// Create the batches
	var line string
	batches := make([][]string, 0, len(imgSubs)/batchMaxRequests+1)
	currentBatch := make([]string, 0, min(batchMaxRequests, len(imgSubs)))
	for id, sub := range imgSubs {
		if line, err = batchCreateLine(id, model, sub.Image, italic); err != nil {
			err = fmt.Errorf("failed to create batch line #%d: %w", id, err)
			return
		}
		if batchSize(currentBatch, line) > batchMaxSize || len(currentBatch) >= batchMaxRequests {
			if debug {
				fmt.Printf("Creating a new batch. Current batch has %d requests for total size of %d",
					len(currentBatch), batchSize(currentBatch, ""))
			}
			batches = append(batches, currentBatch)
			currentBatch = make([]string, 0, min(batchMaxRequests, len(imgSubs)-id))
		}
		currentBatch = append(currentBatch, line)
	}
	batches = append(batches, currentBatch)
	if debug {
		fmt.Printf("Last batch has %d requests for total size of %d\n",
			len(currentBatch), batchSize(currentBatch, ""))
	}
	// Update the files
	uploadedFiles := make([]*openai.FileObject, 0, len(batches))
	defer func() {
		for i, f := range uploadedFiles {
			res, deleteErr := client.Files.Delete(context.TODO(), f.ID)
			if deleteErr != nil {
				fmt.Printf("Failed to delete file %s (batch #%d) after processing: %v\n", f.ID, i, deleteErr)
				continue
			}
			if res.Deleted != true {
				fmt.Printf("File %s (batch #%d) was not deleted after processing\n", f.ID, i)
				continue
			}
			if debug {
				fmt.Printf("Successfully deleted file %s (batch #%d)\n", f.ID, i)
			}
		}
	}()
	var uploadedFile *openai.FileObject
	for i, batch := range batches {
		// TODO marshall
		if uploadedFile, err = client.Files.New(ctx, openai.FileNewParams{
			File:    nil,
			Purpose: openai.FilePurposeBatch,
		}); err != nil {
			err = fmt.Errorf("failed to upload file for batch %d: %w", i, err)
			return
		}
		uploadedFiles = append(uploadedFiles, uploadedFile)
	}
	fmt.Println(len(batches), "batches")
	return
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func batchSize(currentBatch []string, newLine string) (totalSize int) {
	for _, item := range currentBatch {
		totalSize += len(item) + 1 // +1 for the newline character that will be added later in the JSON array representation.
	}
	if newLine != "" {
		totalSize += len(newLine) + 1
	}
	return
}

type BatchLine struct {
	CustomID string                         `json:"custom_id"`
	Method   string                         `json:"method"`
	URL      string                         `json:"url"`
	Body     openai.ChatCompletionNewParams `json:"body"`
}

func batchCreateLine(id int, model string, img image.Image, italic bool) (line string, err error) {
	// Prepare request payload
	body, err := generateOCRBodyRequest(img, model, italic)
	if err != nil {
		err = fmt.Errorf("failed to generate OCR body request: %w", err)
		return
	}
	// Create batch line
	data, err := json.Marshal(BatchLine{
		CustomID: strconv.Itoa(id),
		Method:   "POST",
		URL:      "/v1/chat/completions",
		Body:     body,
	})
	if err != nil {
		err = fmt.Errorf("failed to marshal the batch request line")
		return
	}
	line = string(data)
	return
}
