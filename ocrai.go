package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"image"
	"image/png"

	"github.com/openai/openai-go"
)

const (
	systemPrompt = `Extract the text from the user input. Do not quote, do not say anything but the text.`
)

func OCR(ctx context.Context, imgSubs []*PGSSubtitle, client openai.Client, model string, debug bool) (txtSubs []SRTSubtitle, err error) {
	txtSubs = make([]SRTSubtitle, len(imgSubs))
	var text string
	for index, pg := range imgSubs {
		if text, err = ExtractText(ctx, client, model, pg.Image); err != nil {
			err = fmt.Errorf("Failed to extract text from image #%d: %s\n", index+1, err)
			return
		}
		if debug {
			fmt.Printf("#%d %s --> %s\n%s\n\n", index+1, pg.StartTime, pg.EndTime, text)
		}
		txtSubs[index] = SRTSubtitle{
			Start: SRTTimestamp(pg.StartTime),
			End:   SRTTimestamp(pg.EndTime),
			Text:  text,
		}
	}
	return
}

func ExtractText(ctx context.Context, client openai.Client, model string, img image.Image) (text string, err error) {
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
