package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"image"
	"image/png"
	"os"

	"github.com/hekmon/liveprogress/v2"
	"github.com/openai/openai-go"
)

const (
	systemPrompt = `Extract the text from the user input. Do not quote, do not say anything but the text.`
)

func OCR(ctx context.Context, imgSubs []PGSSubtitle, client openai.Client, model string, debug bool) (txtSubs []SRTSubtitle, err error) {
	// Progress bar
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
		fmt.Printf("VL model tokens used: prompt=%d, completion=%d\n", totalPromptTokens, totalCompletionTokens)
	}()
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
	// Process each subtitle image and extract text using OCR.
	txtSubs = make([]SRTSubtitle, len(imgSubs))
	var (
		text             string
		promptTokens     int64
		completionTokens int64
	)
	for index, pg := range imgSubs {
		if text, promptTokens, completionTokens, err = ExtractText(ctx, client, model, pg.Image); err != nil {
			err = fmt.Errorf("failed to extract text from image #%d: %s\n", index+1, err)
			return
		}
		totalPromptTokens += promptTokens
		totalCompletionTokens += completionTokens
		if debug {
			fmt.Fprintf(bypass, "#%d %s --> %s\n%s\n\n", index+1, pg.StartTime, pg.EndTime, text)
		}
		txtSubs[index] = SRTSubtitle{
			Start: SRTTimestamp(pg.StartTime),
			End:   SRTTimestamp(pg.EndTime),
			Text:  text,
		}
		bar.CurrentIncrement()
	}
	return
}

func ExtractText(ctx context.Context, client openai.Client, model string, img image.Image) (text string, promptTokens, completionTokens int64, err error) {
	// Encode Image
	encodedImage, err := encodeImageToDataURL(img)
	if err != nil {
		err = fmt.Errorf("failed to encode image to data URL: %w", err)
		return
	}
	// Ask model for text extraction
	chatCompletion, err := client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(systemPrompt),
			{
				OfUser: &openai.ChatCompletionUserMessageParam{
					Content: openai.ChatCompletionUserMessageParamContentUnion{
						OfArrayOfContentParts: []openai.ChatCompletionContentPartUnionParam{
							// {
							// 	OfText: &openai.ChatCompletionContentPartTextParam{
							// 		Text: "extract the text from this image",
							// 	},
							// },
							{
								OfImageURL: &openai.ChatCompletionContentPartImageParam{
									ImageURL: openai.ChatCompletionContentPartImageImageURLParam{
										URL: encodedImage,
									},
								},
							},
						},
					},
				},
			},
		},
		Model: model,
	})
	if err != nil {
		err = fmt.Errorf("failed to get OCR chat completion: %w", err)
		return
	}
	text = chatCompletion.Choices[0].Message.Content
	promptTokens = chatCompletion.Usage.PromptTokens
	completionTokens = chatCompletion.Usage.CompletionTokens
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
