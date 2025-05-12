package main

import (
	"errors"
	"image"
	"time"

	"github.com/mbiamont/go-pgs-parser/displaySet"
	"github.com/mbiamont/go-pgs-parser/pgs"
)

type PGSSubtitle struct {
	Image     image.Image
	StartTime time.Duration
	EndTime   time.Duration
}

func ParsePGSFile(filePath string) (subs []*PGSSubtitle, err error) {
	var currentSub *PGSSubtitle
	err = pgs.NewPgsParser().ParseDisplaySets(filePath, func(data displaySet.DisplaySet, startTime time.Duration) error {
		// Check if this display set contains an image or only metadata
		imageData, err := data.ToImageData()
		if err != nil {
			return err
		}
		if imageData != nil {
			// We got a new image so this should be the start of a new sub
			if currentSub != nil {
				return errors.New("got an image without a previous end time for the previous sub")
			}
			currentSub = &PGSSubtitle{
				Image:     imageData.Image,
				StartTime: startTime,
			}
		} else {
			// No image in this display set so it's should be the end of the previous one
			if currentSub == nil {
				return errors.New("got an end time without a previous start time for a previous sub")
			}
			currentSub.EndTime = startTime
			subs = append(subs, currentSub)
			currentSub = nil
		}
		return nil
	})
	return
}
