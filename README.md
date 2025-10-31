# Subtitles AI OCR

Decode Bluray PGS or DVD VobSub subtitles and extract their embedded text using a Vision Language Model (OpenAI API compatible) to create an output SRT file.

## Usage

Once the subtitles are written to the `.srt` file, I recommend you to open it with [Subtitle Edit](https://github.com/SubtitleEdit/subtitleedit) to:

- Check for the model output (if you did not use the `-debug` flag)
- Optimize it with the "Tools > Fix common errors" utility

```raw
Usage of D:\Desktop\subtitles-ai-ocr.exe:
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
  -workers int
        Number of parallel workers. Does nothing with batch mode. (default 1)
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
* Select the variant that fits on your available VRAM of the [Qwen3-VL model](https://ollama.com/library/qwen3-vl).
* For example, with 32GiB of VRAM, I am using the [30B MoE variant](https://ollama.com/library/qwen3-vl:30b)
* If you are VRAM limited you can also check the [8B variant](hhttps://ollama.com/library/qwen3-vl:8b).

#### Linux/MacOS

```bash
# Validate the model and the runtime VRAM needs fits into your available VRAM
# If you are sure it will fits, simply use `pull` instead of `run`
ollama run qwen3-vl:30b
# Run the OCR with the validated model
./subtitles-ai-ocr -baseurl http://127.0.0.1:11434/v1 -timeout 30s -model "qwen3-vl:30b" -input /path/to/input/pgs/subtitle/file.sup -output /path/to/output/subtitle/file.srt -debug
```

#### Windows

Using the command line (`cmd.exe`):

```bat
:: Validate the model and the runtime VRAM needs fits into your available VRAM
:: If you are sure it will fits, simply use `pull` instead of `run`
ollama run qwen3-vl:30b
:: Run the OCR with the validated model
.\subtitles-ai-ocr.exe -baseurl http://127.0.0.1:11434/v1 -timeout 30s -model "qwen3-vl:30b" -input C:\path\to\input\pgs\subtitle\file.sup -output C:\path\to\output\subtitle\file.srt -debug
```

### Advanced (high performance)

If you are going with more advanced deployment with, for example, vLLM and high perf & high quality NVFP4 quantization: you can experiment with the `-workers` flag.

In this example I will be using [Qwen3-VL-30B-A3B-Instruct-NVFP4](https://huggingface.co/ig1/Qwen3-VL-30B-A3B-Instruct-NVFP4) deployed in Docker Desktop with WSL2 and a RTX 5090 (follow the model readme on HF).

Once the model is deployed and vLLM started within Docker Desktop, start the program with 70 workers:

```powershell
C:\Users\hekmon>D:\Desktop\subtitles-ai-ocr.exe -baseurl "http://127.0.0.1:8000/v1" -model "Qwen3-VL-30B-A3B" -timeout 30s -workers 70 -input E:\Vidéo\LSELO\A1_t00_track5_[fre].sub
```

vLLM and NVFP4 will maximize concurrent requests and total throughput:

```text
(APIServer pid=1) INFO 10-30 15:46:40 [loggers.py:208] Engine 000: Avg prompt throughput: 16440.5 tokens/s, Avg generation throughput: 395.6 tokens/s, Running: 55 reqs, Waiting: 0 reqs, GPU KV cache usage: 47.1%, Prefix cache hit rate: 22.8%, MM cache hit rate: 3.8%
(APIServer pid=1) INFO 10-30 15:46:50 [loggers.py:208] Engine 000: Avg prompt throughput: 15850.0 tokens/s, Avg generation throughput: 437.8 tokens/s, Running: 54 reqs, Waiting: 3 reqs, GPU KV cache usage: 47.4%, Prefix cache hit rate: 22.8%, MM cache hit rate: 3.9%
(APIServer pid=1) INFO 10-30 15:47:00 [loggers.py:208] Engine 000: Avg prompt throughput: 14865.2 tokens/s, Avg generation throughput: 408.8 tokens/s, Running: 66 reqs, Waiting: 0 reqs, GPU KV cache usage: 58.5%, Prefix cache hit rate: 22.5%, MM cache hit rate: 3.3%
```

Allowing a thousand subs to be decoded in only 35 seconds:

```text
Parsing VobSub file "A1_t00_track5_[fre].sub"
VobSub file parsed. Total subs: 1057
Qwen3-VL-30B-A3B model tokens used: prompt=547526, completion=14779
OCR completed in 35.0095678s
```

## Going further

Checkout [subtitles-ai-translator](https://github.com/hekmon/subtitles-ai-translator)!

## Thanks

[@mbiamont](https://github.com/mbiamont) and its [go-pgs-parser](https://github.com/mbiamont/go-pgs-parser) library
