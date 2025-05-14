# PGS AI OCR

Extract PGS subtitles and recover text using an external (with OpenAI API) Vision Language model to create an output SRT file.

## Usage

Once the subtitles are written to the `.srt` file, I recommend you to open it with [Subtitle Edit](https://github.com/SubtitleEdit/subtitleedit) to:

- Check for the model output (if you did not use the `-debug` flag)
- Optimize it with the "Tools > Fix common errors" utility

```raw
Usage of ./pgs-ai-ocr:
  -debug
        Print each entry to stdout during the process
  -input string
        PGS file to parse (.sup)
  -italic
        Instruct the model to detect italic text. Not all models manage to do that properly.
  -model string
        AI model to use for OCR. Must be a Vision Language model. (default "o1-mini")
  -output string
        Output subtitle to create (.srt subtitle)
  -timeout duration
        Timeout for the OpenAI API requests (default 30s)
  -version
        show program version
```

### Simple (OpenAI)

#### Linux/MacOS

```bash
export OPENAI_API_KEY="your_openai_api_key_here"
# you can validate with the following cmd: echo $OPENAI_API_KEY
./pgs-ai-ocr -input /path/to/input/pgs/subtitle/file.sup -output /path/to/output/subtitle/file.srt -debug
```

#### Windows

Using the command line (`cmd.exe`):

```bat
set OPENAI_API_KEY=your_openai_api_key_here
:: you can validate with the following cmd: echo %OPENAI_API_KEY%
.\pgs-ai-ocr.exe -italic -input C:\path\to\input\pgs\subtitle\file.sup -output C:\path\to\output\subtitle\file.srt -debug
```

### Advanced (self-hosted)

For local inference [Qwen2.5-VL 7B](https://huggingface.co/Qwen/Qwen2.5-VL-7B-Instruct) is recommended for best results (even if it can not handle properly italic text).

#### Linux/MacOS

```bash
# using vLLM
./pgs-ai-ocr -baseurl http://127.0.0.1:8000/v1 -model "Qwen/Qwen2.5-VL-7B-Instruct" -input /path/to/input/pgs/subtitle/file.sup -output /path/to/output/subtitle/file.srt -debug
```

#### Windows

Using the command line (`cmd.exe`):

```bat
:: using vLLM
.\pgs-ai-ocr.exe -baseurl http://127.0.0.1:8000/v1 -model "Qwen/Qwen2.5-VL-7B-Instruct" -input C:\path\to\input\pgs\subtitle\file.sup -output C:\path\to\output\subtitle\file.srt -debug
```

## Thanks

[@mbiamont](https://github.com/mbiamont) and its [go-pgs-parser](https://github.com/mbiamont/go-pgs-parser) library
