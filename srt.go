package main

import (
	"fmt"
	"io"
	"time"

	"golang.org/x/text/encoding/unicode"
)

type SRTSubtitle struct {
	Start SRTTimestamp
	End   SRTTimestamp
	Text  string
}

type SRTTimestamp time.Duration

func (t SRTTimestamp) String() string {
	return fmt.Sprintf("%02d:%02d:%02d,%03d",
		time.Duration(t)/time.Hour,
		(time.Duration(t)/time.Minute)%60,
		(time.Duration(t)/time.Second)%60,
		time.Duration(t)%time.Second/time.Millisecond,
	)
}

func WriteSRT(output io.Writer, subtitles []SRTSubtitle) (err error) {
	encoder := unicode.UTF8BOM.NewEncoder().Writer(output)
	for i, sub := range subtitles {
		// Num
		if _, err = encoder.Write(fmt.Appendf(nil, "%d\n", i+1)); err != nil {
			return
		}
		// Timestamp
		if _, err = encoder.Write(fmt.Appendf(nil, "%s --> %s\n", sub.Start, sub.End)); err != nil {
			return
		}
		// Text
		if _, err = encoder.Write(fmt.Appendf(nil, "%s\n", sub.Text)); err != nil {
			return
		}
		// Blank line
		if _, err = encoder.Write([]byte("\n")); err != nil {
			return
		}
	}
	return nil
}
