# PGS AI OCR

Extract PGS subtitles and recover text using an external (with OpenAI API) Vision Language model to create an output SRT file.

## Usage

### Simple (OpenAI)

#### Linux/MacOS

```bash
export OAI_API_KEY="your_openai_api_key_here"
# you can validate with the following cmd: echo $OAI_API_KEY
./pgs-ai-ocr -italic -input /path/to/input/pgs/subtitle/file.sup -output /path/to/output/subtitle/file.srt -debug
```

#### Windows

Using the command line (`cmd.exe`):

```bat
set OAI_API_KEY=your_openai_api_key_here
:: you can validate with the following cmd: echo %OAI_API_KEY%
.\pgs-ai-ocr.exe -italic -input C:\path\to\input\pgs\subtitle\file.sup -output C:\path\to\output\subtitle\file.srt -debug
```

### Advanced (self-hosted)

For local inference [Qwen2.5-VL 7B](https://huggingface.co/Qwen/Qwen2.5-VL-7B-Instruct) is recommended for best results (even if it can not handle properly italic text).

#### Linux/MacOS

```bash
export OAI_BASE_URL="http://127.0.0.1:8000/v1" # vLLM endpoint
# you can validate with the following cmd: echo $OAI_BASE_URL
./pgs-ai-ocr -model "Qwen/Qwen2.5-VL-7B-Instruct" -input /path/to/input/pgs/subtitle/file.sup -output /path/to/output/subtitle/file.srt -debug
```

#### Windows

Using the command line (`cmd.exe`):

```bat
set OAI_BASE_URL=http://127.0.0.1:8000/v1
:: you can validate with the following cmd: echo %OAI_BASE_URL%
.\pgs-ai-ocr.exe -model "Qwen/Qwen2.5-VL-7B-Instruct" -input C:\path\to\input\pgs\subtitle\file.sup -output C:\path\to\output\subtitle\file.srt -debug
```
