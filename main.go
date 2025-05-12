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
	APIKEY_ENV  = "OAI_API_KEY"
	BASEURL_ENV = "OAI_BASE_URL"

	//  overrided during compilation
	Version = "dev"
)

func main() {
	// Define flags
	inputPath := flag.String("input", "", "PGS file to parse (.sup)")
	outputPath := flag.String("output", "", "Output subtitle to create (.srt subtitle)")
	model := flag.String("model", openai.ChatModelGPT4o, "AI model to use for OCR. Must be a Vision Language model.")
	italic := flag.Bool("italic", false, "Instruct the model to detect italic text. Not all models manage to do that properly.")
	timeout := flag.Duration("timeout", 30*time.Second, "Timeout for the OpenAI API requests")
	debug := flag.Bool("debug", false, "Print each entry to stdout during the process")
	version := flag.Bool("version", false, "show program version")
	flag.Parse()

	// Checks the flags
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

	// Initiate the openai client
	var err error
	oaiOptions := make([]option.RequestOption, 1, 3)
	oaiOptions[0] = option.WithRequestTimeout(*timeout)
	oaiAPIKey, found := os.LookupEnv(APIKEY_ENV)
	if found {
		oaiOptions = append(oaiOptions, option.WithAPIKey(oaiAPIKey))
	} else {
		fmt.Printf("Environment variable %q not set: OpenAI API client won't be using an API key\n", APIKEY_ENV)
	}
	oaiBaseURL, found := os.LookupEnv(BASEURL_ENV)
	if found {
		if _, err = url.Parse(oaiBaseURL); err != nil {
			fmt.Fprintf(os.Stderr, "Invalid URL for environment variable %q: %s\n", BASEURL_ENV, err.Error())
			return
		}
		fmt.Printf("Environment variable %q set: client will be using a custom base URL: %s\n", BASEURL_ENV, oaiBaseURL)
		oaiOptions = append(oaiOptions, option.WithBaseURL(oaiBaseURL))
	}
	oaiClient := openai.NewClient(oaiOptions...)

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
	start := time.Now()
	srtSubs, err := OCR(runCtx, imgSubs, oaiClient, *model, *italic, *debug)
	if err != nil {
		fmt.Fprintf(os.Stderr, "OCR failed: %s\n", err)
		return
	}
	fmt.Printf("OCR completed in %v\n", time.Since(start))
	if err = WriteSRT(fd, srtSubs); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to write SRT: %s\n", err)
		return
	}
	fmt.Printf("SRT written to %q\n", *outputPath)
}
