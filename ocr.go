package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"image"
	"image/png"
	"os"
	"sync/atomic"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/hekmon/liveprogress/v2"
	"github.com/openai/openai-go/v3"
)

const (
	systemPrompt = `Extract the text from the user's input.
Do not use quotes, do not provide comments, and do not add any additional content beyond the extracted text from the image.
Do not reformulate, write the text exactly as it is on the image even if it is an incomplete sentence. Respect the line breaks.
Maintain the original formatting and line breaks without adding extra spaces or line breaks.
If the user input do not contains any text, simply extract the text from the image. Otherwise consider user instruction as additionnal rules to follow.`

	italicPrompt = `When formatting text, if a single word is in italics, use the following format to mark it:
<i>word</i>

If multiple consecutive words are in italics, use the following format:
non_italic_word_1 <i>italic_word_1 italic_word_2 ... italic_word_n</i> non_italic_word_2

If the italic words span on multiples lines, use the following format:
non_italic_word_1 <i>italic_word_1
italic_word_2 ... italic_word_n</i> non_italic_word_2`
)

type ImageSubtitle struct {
	Image     image.Image
	StartTime time.Duration
	EndTime   time.Duration
}

func OCR(ctx context.Context, imgSubs []ImageSubtitle, nbWorkers int, client openai.Client, model string, italic, debug bool) (txtSubs SRTSubtitles, err error) {
	// Progress bar
	bar := liveprogress.SetMainLineAsBar(
		liveprogress.WithTotal(uint64(len(imgSubs))),
		liveprogress.WithMultiplyRunes(),
		liveprogress.WithPrependDecorator(func(bar *liveprogress.Bar) string {
			return "AI OCR Progress | "
		}),
		liveprogress.WithPrependTimeElapsed(liveprogress.BaseStyle()),
		liveprogress.WithAppendPercent(liveprogress.BaseStyle()),
		liveprogress.WithAppendDecorator(func(bar *liveprogress.Bar) string {
			return fmt.Sprintf(" | %d/%d images processed | ETA:", bar.Current(), bar.Total())
		}),
		liveprogress.WithAppendTimeRemaining(liveprogress.BaseStyle()),
	)
	defer liveprogress.RemoveBar(bar)
	bypass := liveprogress.Bypass()
	if err = liveprogress.Start(); err != nil {
		err = fmt.Errorf("failed to start live progress: %w", err)
		return
	}
	// End log
	var (
		totalPromptTokens     atomic.Int64
		totalCompletionTokens atomic.Int64
	)
	defer func() {
		var clear bool
		if err == nil {
			clear = true
		}
		if err := liveprogress.Stop(clear); err != nil {
			fmt.Fprintf(os.Stderr, "failed to stop live progress: %s\n", err)
		}
		fmt.Printf("%q model tokens used: prompt=%d, completion=%d\n",
			model, totalPromptTokens.Load(), totalCompletionTokens.Load(),
		)
	}()
	// Process each subtitle image and extract text using AI OCR
	//// feeder worker
	type TodoJob struct {
		Index    int
		Subtitle ImageSubtitle
	}
	todoJobs := make(chan TodoJob)
	feederCtx, feederCtxCancel := context.WithCancel(ctx)
	defer feederCtxCancel()
	go func() {
		defer close(todoJobs)
		for index, sub := range imgSubs {
			select {
			case todoJobs <- TodoJob{
				Index:    index,
				Subtitle: sub,
			}:
			case <-feederCtx.Done():
				return
			}
		}
	}()
	//// ocr workers
	txtSubs = make(SRTSubtitles, len(imgSubs))
	ocrWorkers := new(errgroup.Group)
	for range nbWorkers {
		ocrWorkers.Go(func() (err error) {
			var (
				text             string
				promptTokens     int64
				completionTokens int64
			)
			for job := range todoJobs {
				text, promptTokens, completionTokens, err = ExtractText(ctx, client, model, job.Subtitle.Image, italic)
				if err != nil {
					err = fmt.Errorf("failed to extract text from image #%d: %s\n", job.Index+1, err)
					return
				}
				txtSubs[job.Index] = SRTSubtitle{
					Start: SRTTimestamp(job.Subtitle.StartTime),
					End:   SRTTimestamp(job.Subtitle.EndTime),
					Text:  text,
				}
				totalPromptTokens.Add(promptTokens)
				totalCompletionTokens.Add(completionTokens)
				bar.CurrentIncrement()
				if debug {
					fmt.Fprintf(bypass, "#%d %s --> %s\n%s\n\n",
						job.Index+1, job.Subtitle.StartTime, job.Subtitle.EndTime, text,
					)
				}
			}
			return
		})
	}
	if err = ocrWorkers.Wait(); err != nil {
		feederCtxCancel() // stop the feeder worker early if a worker encountered an error
		err = fmt.Errorf("a worker encountered an error: %w", err)
		return
	}
	return
}

func ExtractText(ctx context.Context, client openai.Client, model string, img image.Image, italic bool) (text string, promptTokens, completionTokens int64, err error) {
	// Prepare payload
	body, err := generateOCRBodyRequest(img, model, italic)
	if err != nil {
		err = fmt.Errorf("failed to generate OCR body request: %w", err)
		return
	}
	// Ask model for text extraction
	chatCompletion, err := client.Chat.Completions.New(ctx, body)
	if err != nil {
		err = fmt.Errorf("failed to get OCR chat completion: %w", err)
		return
	}
	text = chatCompletion.Choices[0].Message.Content
	promptTokens = chatCompletion.Usage.PromptTokens
	completionTokens = chatCompletion.Usage.CompletionTokens
	return
}

func generateOCRBodyRequest(img image.Image, model string, italic bool) (body openai.ChatCompletionNewParams, err error) {
	encodedImage, err := encodeImageToDataURL(img)
	if err != nil {
		err = fmt.Errorf("failed to encode image: %w", err)
		return
	}
	content := make([]openai.ChatCompletionContentPartUnionParam, 0, 2)
	if italic {
		// Set the additionnal italic instructions as user prompt
		content = append(content, openai.ChatCompletionContentPartUnionParam{
			OfText: &openai.ChatCompletionContentPartTextParam{
				Text: italicPrompt,
			},
		})
	}
	content = append(content, openai.ChatCompletionContentPartUnionParam{
		OfImageURL: &openai.ChatCompletionContentPartImageParam{
			ImageURL: openai.ChatCompletionContentPartImageImageURLParam{
				URL: encodedImage,
			},
		},
	})
	body = openai.ChatCompletionNewParams{
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(systemPrompt),
			{
				OfUser: &openai.ChatCompletionUserMessageParam{
					Content: openai.ChatCompletionUserMessageParamContentUnion{
						OfArrayOfContentParts: content,
					},
				},
			},
		},
		Model: model,
		// Temperature: param.Opt[float64]{
		// 	Value: temperature,
		// },
	}
	return
}

func encodeImageToDataURL(image image.Image) (string, error) {
	var data bytes.Buffer
	err := png.Encode(&data, image)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("data:image/png;base64,%s", base64.StdEncoding.EncodeToString(data.Bytes())), nil
}
