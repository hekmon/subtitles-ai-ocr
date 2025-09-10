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

	// overrided during compilation
	Version = "dev"
)

func main() {
	// Define flags
	inputPath := flag.String("input", "", "Image subtitles file to decode (.sup for Bluray PGS and .sub -.idx must also be present- for DVD VobSub)")
	outputPath := flag.String("output", "", "Output subtitle to create (.srt subtitle). Default will use same folder and same filename as input but with .srt extension")
	baseURL := flag.String("baseurl", OAI_BASEURL, "OpenAI API base URL")
	model := flag.String("model", "gpt-5-nano-2025-08-07", "AI model to use for OCR. Must be a Vision Language Model.")
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
	} else if !strings.HasSuffix(*inputPath, ".sup") && !strings.HasSuffix(*inputPath, ".sub") {
		fmt.Fprintf(os.Stderr, "The input file must be a .sup (Bluray PGS) or .sub (DVD VobSub) file\n")
		return
	}
	if *outputPath == "" {
		*outputPath = filepath.Join(filepath.Dir(*inputPath), strings.TrimSuffix(filepath.Base(*inputPath), filepath.Ext(*inputPath))+".srt")
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

	// Step 1 - Parse subtitle file
	var subs map[int][]ImageSubtitle
	if strings.HasSuffix(*inputPath, ".sup") {
		fmt.Printf("Parsing PGS file %q\n", filepath.Base(*inputPath))
		imgSubs, err := ParsePGSFile(*inputPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to parse PGS file: %s\n", err)
			return
		}
		if *debug {
			for _, sub := range imgSubs {
				fmt.Printf("Start: %v, End: %v, Size: %d×%v\n",
					sub.StartTime, sub.EndTime, sub.Image.Bounds().Dx(), sub.Image.Bounds().Dy(),
				)
			}
		}
		fmt.Println("PGS file parsed. Total subs:", len(imgSubs))
		if len(imgSubs) == 0 {
			return
		}
		subs = map[int][]ImageSubtitle{
			0: imgSubs,
		}
	} else {
		fmt.Printf("Parsing VobSub file %q\n", filepath.Base(*inputPath))
		if subs, err = ParseVobSubFile(*inputPath); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to parse PGS file: %s\n", err)
			return
		}
		var (
			subIndex  string
			totalSubs int
		)
		for index, imgSubs := range subs {
			if len(subs) > 1 {
				subIndex = fmt.Sprintf("[%d] ", index)
			}
			totalSubs += len(imgSubs)
			if *debug {
				for _, sub := range imgSubs {
					fmt.Printf("%sStart: %v, End: %v, Size: %d×%v\n",
						subIndex, sub.StartTime, sub.EndTime,
						sub.Image.Bounds().Dx(), sub.Image.Bounds().Dy(),
					)
				}
			}
		}
		if len(subs) > 1 {
			subIndex = fmt.Sprintf(" (over %d streams)", len(subs))
		}
		fmt.Printf("VobSub file parsed. Total subs: %d%s\n", totalSubs, subIndex)
		if totalSubs == 0 {
			return
		}
	}

	// Prepare clean stop
	runCtx, runCtxStopFunc := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer runCtxStopFunc()

	// Step 2 - OCR with AI
	for streamID, streamSubs := range subs {
		var finalOutputPath string
		// Adjust outputpath if needed
		if len(subs) > 1 {
			dirPath := filepath.Dir(*outputPath)
			file := filepath.Base(*outputPath)
			extension := filepath.Ext(file)
			fileName := file[:len(file)-len(extension)]
			finalOutputPath = filepath.Join(dirPath, fmt.Sprintf("%s_stream-%d%s", fileName, streamID, extension))
			fmt.Printf("Stream #%d\n", streamID)
		} else {
			finalOutputPath = *outputPath
		}
		// Start process
		if err = processSubsImages(runCtx, streamSubs, oaiClient, *model, finalOutputPath, *batchMode, *italic, *debug); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to write the output test file: %s\n", err)
			return
		}
		if finalOutputPath != *outputPath {
			fmt.Println()
		}
	}
}

func processSubsImages(ctx context.Context, imgSubs []ImageSubtitle, oaiClient openai.Client, model, outputPath string,
	batch, italic, debug bool) (err error) {
	// Check if we can create the output file now to avoid loosing the extraction if we can not save it afterwards
	var fd *os.File
	if fd, err = os.Create(outputPath); err != nil {
		err = fmt.Errorf("Failed to write the output test file: %w", err)
		return
	}
	defer fd.Close()
	// Prepare OCR via AI
	liveprogress.RefreshInterval = 500 * time.Millisecond
	var srtSubs SRTSubtitles
	start := time.Now()
	if batch {
		if srtSubs, err = OCRBatched(ctx, imgSubs, oaiClient, model, italic, debug); err != nil {
			err = fmt.Errorf("batched OCR failed: %w", err)
			return
		}
	} else {
		if srtSubs, err = OCR(ctx, imgSubs, oaiClient, model, italic, debug); err != nil {
			err = fmt.Errorf("OCR failed: %w", err)
			return
		}
	}
	fmt.Printf("OCR completed in %v\n", time.Since(start))
	// Step 3 - Write SRT file
	if err = srtSubs.Marshal(fd); err != nil {
		err = fmt.Errorf("failed to write SRT: %s\n", err)
		return
	}
	fmt.Printf("SRT written to %q\n", outputPath)
	return
}
