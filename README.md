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
* If you are VRAM limited you can also check the [8B variant](https://ollama.com/library/qwen3-vl:8b).

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

It has a high loading time in order to be highly performant during inference, use it when you have a batch of files to process.

Once the model is deployed and vLLM started within Docker Desktop, start the program with 70 workers and loop thru all the files that needs to be processed (example in powershell):

```powershell
for ($i = 1; $i -le 24; $i++) {
    $episode = "e{0:d2}" -f $i
    $inputFile = "E:\Vidéo\Jujutsu Kaisen S01\${episode}_track3_[fre].sup"
    
    & "D:\Desktop\subtitles-ai-ocr.exe" -baseurl "http://127.0.0.1:8000/v1" -model "Qwen3-VL-30B-A3B" -timeout 60s -workers 70 -input "E:\Vidéo\Jujutsu Kaisen S01\${episode}_track3_[fre].sup"
} 
```

vLLM and NVFP4 will maximize concurrent requests and total throughput:

```text
[loggers.py:248] Engine 000: Avg prompt throughput: 17015.0 tokens/s, Avg generation throughput: 889.6 tokens/s, Running: 52 reqs, Waiting: 0 reqs, GPU KV cache usage: 11.5%, Prefix cache hit rate: 53.6%, MM cache hit rate: 9.0%
[loggers.py:248] Engine 000: Avg prompt throughput: 17060.3 tokens/s, Avg generation throughput: 894.1 tokens/s, Running: 61 reqs, Waiting: 0 reqs, GPU KV cache usage: 13.9%, Prefix cache hit rate: 53.6%, MM cache hit rate: 3.7%
[loggers.py:248] Engine 000: Avg prompt throughput: 15035.4 tokens/s, Avg generation throughput: 836.8 tokens/s, Running: 35 reqs, Waiting: 1 reqs, GPU KV cache usage: 5.8%, Prefix cache hit rate: 53.6%, MM cache hit rate: 5.7%
[loggers.py:248] Engine 000: Avg prompt throughput: 16897.8 tokens/s, Avg generation throughput: 878.3 tokens/s, Running: 59 reqs, Waiting: 0 reqs, GPU KV cache usage: 12.2%, Prefix cache hit rate: 53.6%, MM cache hit rate: 5.7%
[loggers.py:248] Engine 000: Avg prompt throughput: 17367.0 tokens/s, Avg generation throughput: 925.5 tokens/s, Running: 15 reqs, Waiting: 0 reqs, GPU KV cache usage: 4.1%, Prefix cache hit rate: 54.9%, MM cache hit rate: 7.5%
[loggers.py:248] Engine 000: Avg prompt throughput: 14996.4 tokens/s, Avg generation throughput: 808.3 tokens/s, Running: 21 reqs, Waiting: 0 reqs, GPU KV cache usage: 3.7%, Prefix cache hit rate: 53.2%, MM cache hit rate: 4.6%
[loggers.py:248] Engine 000: Avg prompt throughput: 17703.6 tokens/s, Avg generation throughput: 930.9 tokens/s, Running: 56 reqs, Waiting: 0 reqs, GPU KV cache usage: 12.3%, Prefix cache hit rate: 53.1%, MM cache hit rate: 3.6%
[loggers.py:248] Engine 000: Avg prompt throughput: 15815.0 tokens/s, Avg generation throughput: 831.5 tokens/s, Running: 61 reqs, Waiting: 0 reqs, GPU KV cache usage: 14.2%, Prefix cache hit rate: 53.4%, MM cache hit rate: 4.1%
[loggers.py:248] Engine 000: Avg prompt throughput: 16363.9 tokens/s, Avg generation throughput: 869.3 tokens/s, Running: 60 reqs, Waiting: 0 reqs, GPU KV cache usage: 12.6%, Prefix cache hit rate: 53.4%, MM cache hit rate: 3.9%
```

Allowing a complete season to be processed in less than 3 minutes:

```text
[...]
Parsing PGS file "e24_track3_[fre].sup"
PGS file parsed. Total subs: 375
OCR completed in 4.232s
"Qwen3-VL-30B-A3B" model statistics:
        prompt tokens:     79846 (~18868 tokens/s)
        generation tokens: 4270 (~1009 tokens/s)
SRT written to "E:\\Vidéo\\Jujutsu Kaisen S01\\e24_track3_[fre].srt"
```

## Going further

Checkout [subtitles-ai-translator](https://github.com/hekmon/subtitles-ai-translator)!

## Thanks

[@mbiamont](https://github.com/mbiamont) and its [go-pgs-parser](https://github.com/mbiamont/go-pgs-parser) library
