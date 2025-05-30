package main

import (
	"context"
	"flag"
	"fmt"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/hekmon/liveprogress/v2"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

const (
	APIKEY_ENV  = "OPENAI_API_KEY" // default from openai client
	OAI_BASEURL = "https://api.openai.com/v1"

	//  overrided during compilation
	Version = "dev"
)

func main() {
	// Define flags
	inputPath := flag.String("input", "", "PGS file to parse (.sup)")
	outputPath := flag.String("output", "", "Output subtitle to create (.srt subtitle)")
	baseURL := flag.String("baseurl", OAI_BASEURL, "OpenAI API base URL")
	model := flag.String("model", "gpt-4.1-nano-2025-04-14", "AI model to use for OCR. Must be a Vision Language model.")
	italic := flag.Bool("italic", false, "Instruct the model to detect italic text. So far no models managed to detect it properly.")
	batchMode := flag.Bool("batch", false, "OpenAI batch mode. Longer (up to 24h) but cheaper (-50%). You should validate a few samples in regular mode first.")
	timeout := flag.Duration("timeout", 10*time.Minute, "Timeout for the OpenAI API requests")
	debug := flag.Bool("debug", false, "Print each entry to stdout during the process")
	version := flag.Bool("version", false, "show program version")
	flag.Parse()

	// Checks params
	if *version {
		fmt.Printf("Version: %s\n", Version)
		return
	}
	if *inputPath == "" {
		fmt.Fprintf(os.Stderr, "Please set the -input flag\n\n")
		flag.Usage()
		return
	} else if !strings.HasSuffix(*inputPath, ".sup") {
		fmt.Fprintf(os.Stderr, "The input file must be a .sup file\n")
		return
	}
	if *outputPath == "" {
		fmt.Fprintf(os.Stderr, "Please set the -output flag\n\n")
		flag.Usage()
		return
	} else if !strings.HasSuffix(*outputPath, ".srt") {
		fmt.Fprintf(os.Stderr, "The output file must be a .srt file\n")
		return
	}
	if *baseURL != OAI_BASEURL && *batchMode {
		fmt.Fprintf(os.Stderr, "Batch mode is not supported for custom base URLs.\n")
		return
	}
	var err error
	if _, err := url.Parse(*baseURL); err != nil {
		fmt.Fprintf(os.Stderr, "Invalid URL for -baseURL flag: %s\n", err.Error())
		return
	}
	if _, found := os.LookupEnv(APIKEY_ENV); !found {
		fmt.Printf("Environment variable %q not set: OpenAI API client won't be using an API key\n", APIKEY_ENV)
	}

	// Initiate the openai client (default options will automatically lookup for OPENAI_API_KEY env var)
	oaiClient := openai.NewClient(
		option.WithRequestTimeout(*timeout),
		option.WithBaseURL(*baseURL),
	)

	// Check if we can create the output file now to avoid loosing the extraction if we can not save it afterwards
	var fd *os.File
	if fd, err = os.Create(*outputPath); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to write the output test file: %s\n", err)
		return
	}
	defer fd.Close()

	// Step 1 - Parse PGS file
	fmt.Printf("Parsing PGS file %q\n", filepath.Base(*inputPath))
	imgSubs, err := ParsePGSFile(*inputPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to parse PGS file: %s\n", err)
		return
	}
	if *debug {
		for _, sub := range imgSubs {
			fmt.Printf("Start: %v, End: %v, Size: %d×%v\n",
				sub.StartTime, sub.EndTime, sub.Image.Bounds().Dx(), sub.Image.Bounds().Dy())
		}
	}
	fmt.Println("PGS file parsed. Total subs:", len(imgSubs))

	// Prepare clean stop
	runCtx, runCtxStopFunc := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer runCtxStopFunc()

	// Step 2 - OCR with AI
	liveprogress.RefreshInterval = 500 * time.Millisecond
	var srtSubs SRTSubtitles
	start := time.Now()
	if *batchMode {
		if srtSubs, err = OCRBatched(runCtx, imgSubs, oaiClient, *model, *italic, *debug); err != nil {
			fmt.Fprintf(os.Stderr, "batched OCR failed: %s\n", err)
			return
		}
	} else {
		if srtSubs, err = OCR(runCtx, imgSubs, oaiClient, *model, *italic, *debug); err != nil {
			fmt.Fprintf(os.Stderr, "OCR failed: %s\n", err)
			return
		}
	}
	fmt.Printf("OCR completed in %v\n", time.Since(start))

	// Step 3 - Write SRT file
	if err = srtSubs.Marshal(fd); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to write SRT: %s\n", err)
		return
	}
	fmt.Printf("SRT written to %q\n", *outputPath)
}
