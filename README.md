# Subtitles AI OCR

Decode Bluray PGS or DVD VobSub subtitles and extract their embedded text using an external (with OpenAI API) Vision Language Model to create an output SRT file.

## Usage

Once the subtitles are written to the `.srt` file, I recommend you to open it with [Subtitle Edit](https://github.com/SubtitleEdit/subtitleedit) to:

- Check for the model output (if you did not use the `-debug` flag)
- Optimize it with the "Tools > Fix common errors" utility

```raw
Usage of ./subtitles-ai-ocr:
  -baseurl string
        OpenAI API base URL (default "https://api.openai.com/v1")
  -batch
        OpenAI batch mode. Longer (up to 24h) but cheaper (-50%). The progress bar won't help you much: it will be ready when it will be ready! You should validate a few samples in regular mode first.
  -debug
        Print each entry to stdout during the process
  -input string
        Image subtitles file to decode (.sup for Bluray PGS and .sub -.idx must also be present- for DVD VobSub)
  -italic
        Instruct the model to detect italic text. So far no models managed to detect it properly.
  -model string
        AI model to use for OCR. Must be a Vision Language Model. (default "gpt-5-nano-2025-08-07")
  -output string
        Output subtitle to create (.srt subtitle). Default will use same folder and same filename as input but with .srt extension
  -timeout duration
        Timeout for the OpenAI API requests (default 10m0s)
  -version
        show program version
```

### Simple (OpenAI)

#### Linux/MacOS

```bash
export OPENAI_API_KEY="your_openai_api_key_here"
# you can validate with the following cmd: echo $OPENAI_API_KEY
./subtitles-ai-ocr -input /path/to/input/pgs/subtitle/file.sup -debug
```

#### Windows

Using the command line (`cmd.exe`):

```bat
set OPENAI_API_KEY=your_openai_api_key_here
:: you can validate with the following cmd: echo %OPENAI_API_KEY%
.\subtitles-ai-ocr.exe -input C:\path\to\input\pgs\subtitle\file.sup -debug
```

### Advanced (OpenAI batch mode)

You will loose the progress bar usefullness and the streaming debug but it will cost you half the regular price.
You should validate a few lines without batch mode and with the `-debug` flag first and then process the entire file with batch mode to cut cost.
This example also added the `-output` flag for custom output file path but it's optional.

#### Linux/MacOS

```bash
export OPENAI_API_KEY="your_openai_api_key_here"
# you can validate with the following cmd: echo $OPENAI_API_KEY
./subtitles-ai-ocr -input /path/to/input/pgs/subtitle/file.sup -output /path/to/output/subtitle/file.srt -batch
```

#### Windows

Using the command line (`cmd.exe`):

```bat
set OPENAI_API_KEY=your_openai_api_key_here
:: you can validate with the following cmd: echo %OPENAI_API_KEY%
.\subtitles-ai-ocr.exe -input C:\path\to\input\pgs\subtitle\file.sup -output C:\path\to\output\subtitle\file.srt -batch
```

### Advanced (self-hosted)

This example will use Ollama but you can use any self hosted inference server as long as it has an OpenAI API compatible endpoint.

#### Pre-requisites

* Install [Ollama](https://ollama.com/).
* Select the variant that fits on your available VRAM of the [Qwen2.5-VL model](https://ollama.com/library/qwen2.5vl).
* For example, with 32GiB of VRAM, I am using the [32B variant](https://ollama.com/library/qwen2.5vl:32b) (28GiB usage for 4k context, 31,5GiB with 32k context).
* If you are VRAM limited you can also check the [7B variant](https://ollama.com/library/qwen2.5vl:7b).

#### Linux/MacOS

```bash
# Validate the model and the runtime VRAM needs fits into your available VRAM
# If you are sure it will fits, simply use `pull` instead of `run`
ollama run qwen2.5vl:32b
# Run the OCR with the validated model
./subtitles-ai-ocr -baseurl http://127.0.0.1:11434/v1 -timeout 30s -model "qwen2.5vl:32b" -input /path/to/input/pgs/subtitle/file.sup -output /path/to/output/subtitle/file.srt -debug
```

#### Windows

Using the command line (`cmd.exe`):

```bat
:: Validate the model and the runtime VRAM needs fits into your available VRAM
:: If you are sure it will fits, simply use `pull` instead of `run`
ollama run qwen2.5vl:32b
:: Run the OCR with the validated model
.\subtitles-ai-ocr.exe -baseurl http://127.0.0.1:11434/v1 -timeout 30s -model "qwen2.5vl:32b" -input C:\path\to\input\pgs\subtitle\file.sup -output C:\path\to\output\subtitle\file.srt -debug
```

## Going further

Checkout [subtitles-ai-translator](https://github.com/hekmon/subtitles-ai-translator)!

## Thanks

[@mbiamont](https://github.com/mbiamont) and its [go-pgs-parser](https://github.com/mbiamont/go-pgs-parser) library
