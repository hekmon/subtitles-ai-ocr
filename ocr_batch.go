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
	var line string
	for id, sub := range imgSubs {
		if line, err = batchCreateLine(id, model, sub.Image, italic); err != nil {
			err = fmt.Errorf("failed to create batch line #%d: %w", id, err)
			return
		}
		if debug {
			fmt.Println(line)
		}
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
	encodedImage, err := encodeImageToDataURL(img)
	if err != nil {
		err = fmt.Errorf("failed to encode image: %w", err)
		return
	}
	content := make([]openai.ChatCompletionContentPartUnionParam, 0, 2)
	if italic {
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
	data, err := json.Marshal(BatchLine{
		CustomID: strconv.Itoa(id),
		Method:   "POST",
		URL:      "/v1/chat/completions",
		Body: openai.ChatCompletionNewParams{
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
		},
	})
	if err != nil {
		err = fmt.Errorf("failed to marshal the batch request line")
		return
	}
	line = string(data)
	return
}
