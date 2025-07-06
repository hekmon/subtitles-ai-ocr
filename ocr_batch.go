package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/hekmon/liveprogress/v2"
	"github.com/openai/openai-go"
)

const (
	batchMaxRequests = 50_000
	batchMaxSize     = 200_000_000 // 200 MB
	batchMaxWaitTime = openai.BatchNewParamsCompletionWindow24h
)

type batchContent []string

func (bc batchContent) Reader() (r io.Reader, err error) {
	var fileContent bytes.Buffer
	for _, line := range bc {
		if _, err = fileContent.WriteString(line); err != nil {
			return
		}
		if _, err = fileContent.WriteRune('\n'); err != nil {
			return
		}
	}
	r = &fileContent
	return
}

func OCRBatched(ctx context.Context, imgSubs []PGSSubtitle, client openai.Client, model string, italic, debug bool) (txtSubs SRTSubtitles, err error) {
	// Create the batches
	var line string
	batches := make([]batchContent, 0, len(imgSubs)/batchMaxRequests+1)
	currentBatch := make(batchContent, 0, min(batchMaxRequests, len(imgSubs)))
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
			currentBatch = make(batchContent, 0, min(batchMaxRequests, len(imgSubs)-id))
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
			if !res.Deleted {
				fmt.Printf("File %s (batch #%d) was not deleted after processing\n", f.ID, i)
				continue
			}
			if debug {
				fmt.Printf("Successfully deleted file %s (batch #%d)\n", f.ID, i)
			}
		}
	}()
	var (
		uploadedFile *openai.FileObject
		reader       io.Reader
	)
	for i, batch := range batches {
		if reader, err = batch.Reader(); err != nil {
			err = fmt.Errorf("failed to create reader for batch #%d: %w", i, err)
			return
		}
		if uploadedFile, err = client.Files.New(ctx, openai.FileNewParams{
			File:    reader,
			Purpose: openai.FilePurposeBatch,
		}); err != nil {
			err = fmt.Errorf("failed to upload file for batch %d: %w", i, err)
			return
		}
		uploadedFiles = append(uploadedFiles, uploadedFile)
		if debug {
			fmt.Printf("Successfully uploaded %q (batch #%d)\n", uploadedFile.ID, i)
		}
	}
	// Schedule the batches
	maxWaitDuration, err := time.ParseDuration(string(batchMaxWaitTime))
	if err != nil {
		err = fmt.Errorf("failed to parse completion window duration: %w", err)
		return
	}
	var res *openai.Batch
	scheduledBatches := make([]*openai.Batch, 0, len(uploadedFiles))
	for batchIndex, batchFile := range uploadedFiles {
		if res, err = client.Batches.New(ctx, openai.BatchNewParams{
			CompletionWindow: batchMaxWaitTime,
			Endpoint:         openai.BatchNewParamsEndpointV1ChatCompletions,
			InputFileID:      batchFile.ID,
		}); err != nil {
			err = fmt.Errorf("failed to schedule batch for file %s: %w", batchFile.ID, err)
			return
		}
		scheduledBatches = append(scheduledBatches, res)
		if debug {
			fmt.Printf("Successfully scheduled %q for file %q (batch #%d)\n", res.ID, batchFile.ID, batchIndex)
		}
	}
	start := time.Now()
	maxEndTime := start.Add(maxWaitDuration)
	// Prepare progress bar
	var (
		totalPromptTokens     int64
		totalCompletionTokens int64
	)
	if err = liveprogress.Start(); err != nil {
		err = fmt.Errorf("failed to start live progress: %w", err)
		return
	}
	defer func() {
		var clear bool
		if err == nil {
			clear = true
		}
		if err := liveprogress.Stop(clear); err != nil {
			fmt.Fprintf(os.Stderr, "failed to stop live progress: %s\n", err)
		}
		fmt.Printf("%s model tokens used: prompt=%d, completion=%d\n", model, totalPromptTokens, totalCompletionTokens)
	}()
	bar := liveprogress.SetMainLineAsCustomLine(func() string {
		var nbOK int
		for _, batch := range scheduledBatches {
			if batch.Status == openai.BatchStatusCompleted {
				nbOK++
			}
		}
		var suffix string
		if len(scheduledBatches) > 1 {
			suffix = "es"
		}
		return fmt.Sprintf("%s | %d/%d batch%s completed | %s",
			time.Since(start).Truncate(time.Second),
			nbOK, len(scheduledBatches), suffix,
			time.Since(maxEndTime).Truncate(time.Second)*-1,
		)
	})
	defer liveprogress.RemoveCustomLine(bar)
	bypass := liveprogress.Bypass()
	// Wait
	check := time.NewTicker(time.Minute)
	defer check.Stop()
	previousBatchesStatus := make([]openai.BatchStatus, len(scheduledBatches))
	defer func() {
		// Cancel running batch if we are exiting early
		if err == nil {
			// if we are exiting normally, nothing to do
			return
		}
		for batchIndex, batch := range scheduledBatches {
			switch batch.Status {
			case openai.BatchStatusFailed, openai.BatchStatusCompleted, openai.BatchStatusCancelling, openai.BatchStatusCancelled:
				continue // no need to cancel non running batches
			default:
				if res, err := client.Batches.Cancel(context.TODO(), batch.ID); err != nil {
					fmt.Fprintf(bypass, "Batch #%d (ID: %q) cancelling failed: %s\n",
						batchIndex, batch.ID, err.Error())
				} else {
					fmt.Fprintf(bypass, "Batch #%d (ID: %q) cancelling new status: %s\n",
						batchIndex, batch.ID, res.Status)
				}
			}
		}
	}()
waitLoop:
	for {
		select {
		case <-check.C:
			for batchIndex, batch := range scheduledBatches {
				if scheduledBatches[batchIndex], err = client.Batches.Get(ctx, batch.ID); err != nil {
					err = fmt.Errorf("failed to get status of batch %s: %w", batch.ID, err)
					return
				}
				if previousBatchesStatus[batchIndex] == scheduledBatches[batchIndex].Status {
					continue
				}
				if debug {
					fmt.Fprintf(bypass, "Batch #%d (ID: %q) status changed from %q to %q\n",
						batchIndex, batch.ID, previousBatchesStatus[batchIndex], batch.Status)
				}
				previousBatchesStatus[batchIndex] = scheduledBatches[batchIndex].Status
				switch scheduledBatches[batchIndex].Status {
				case openai.BatchStatusValidating:
					// continue to wait
					continue
				case openai.BatchStatusFailed:
					err = fmt.Errorf("batch %s failed: %s", batch.ID, batch.Errors.RawJSON())
					return
				case openai.BatchStatusInProgress:
					// continue to wait
					continue
				case openai.BatchStatusFinalizing:
					// continue to wait
					continue
				case openai.BatchStatusCompleted:
					// continue to the next check
				case openai.BatchStatusExpired:
					err = fmt.Errorf("batch %s expired", batch.ID)
					return
				case openai.BatchStatusCancelling:
					// continue to wait
					continue
				case openai.BatchStatusCancelled:
					err = fmt.Errorf("batch %s cancelled", batch.ID)
					return
				default:
					fmt.Fprintf(bypass, "Unknown batch status for batch %s: %s\n", batch.ID, batch.Status)
					continue
				}
				allDone := true
				for _, batch := range scheduledBatches {
					if batch.Status != openai.BatchStatusCompleted {
						allDone = false
						break
					}
				}
				if allDone {
					break waitLoop
				}
			}
		case <-ctx.Done():
			err = ctx.Err()
			return
		}
	}
	fmt.Fprintf(bypass, "All batches completed in %s.\n", time.Since(start))
	// Get the results
	defer func() {
		// defer delete results file
		for batchIndex, batch := range scheduledBatches {
			// Get content
			if res, err := client.Files.Delete(context.TODO(), batch.OutputFileID); err != nil {
				fmt.Fprintf(bypass, "Batch #%d (ID: %q) results file ID %q deletion failed: %s\n",
					batchIndex, batch.ID, batch.OutputFileID, err.Error())
			} else {
				fmt.Fprintf(bypass, "Batch #%d (ID: %q) results file ID %q deletion: %s\n",
					batchIndex, batch.ID, batch.OutputFileID, res.Deleted)
			}
		}
	}()
	txtSubs = make(SRTSubtitles, len(imgSubs))
	var (
		results  *http.Response
		subIndex int
	)
	for batchIndex, batch := range scheduledBatches {
		// Get content
		if results, err = client.Files.Content(ctx, batch.OutputFileID); err != nil {
			err = fmt.Errorf("Failed to get batch #%d result file (ID %s): %w",
				batchIndex, batch.OutputFileID, err,
			)
			return
		}
		// Scan line by line
		scanner := bufio.NewScanner(results.Body)
		for scanner.Scan() {
			var line BatchLineResponse
			if err = json.Unmarshal([]byte(scanner.Text()), &line); err != nil {
				err = fmt.Errorf("Failed to unmarshal batch #%d result line: %w", batchIndex, err)
				return
			}
			// Process the line
			if subIndex, err = strconv.Atoi(line.CustomID); err != nil {
				err = fmt.Errorf("Failed to convert custom ID to integer: %w", err)
				return
			}
			txtSubs[subIndex] = SRTSubtitle{
				Start: SRTTimestamp(imgSubs[subIndex].StartTime),
				End:   SRTTimestamp(imgSubs[subIndex].EndTime),
				Text:  line.Response.Body.Choices[0].Message.Content,
			}
			totalPromptTokens += line.Response.Body.Usage.PromptTokens
			totalCompletionTokens += line.Response.Body.Usage.CompletionTokens
			if debug {
				fmt.Fprintf(bypass, "#%d %s --> %s (batch #%d)\n%s\n\n",
					subIndex+1, imgSubs[subIndex].StartTime, imgSubs[subIndex].EndTime, batchIndex,
					line.Response.Body.Choices[0].Message.Content,
				)
			}
		}
		if err = scanner.Err(); err != nil {
			err = fmt.Errorf("Failed to scan batch #%d result file: %w", batchIndex, err)
			return
		}
	}
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

type batchLine struct {
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
	data, err := json.Marshal(batchLine{
		CustomID: strconv.Itoa(id),
		Method:   "POST",
		URL:      string(openai.BatchNewParamsEndpointV1ChatCompletions),
		Body:     body,
	})
	if err != nil {
		err = fmt.Errorf("failed to marshal the batch request line")
		return
	}
	line = string(data)
	return
}

type BatchLineResponse struct {
	ID       string `json:"id"`
	CustomID string `json:"custom_id"`
	Response struct {
		StatusCode int                   `json:"status_code"`
		RequestID  string                `json:"request_id"`
		Body       openai.ChatCompletion `json:"body"`
	} `json:"response"`
	Error interface{} `json:"error"`
}
