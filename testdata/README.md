# Commands to Build test.mkv

This document records all the `mkvmerge` commands used to build the `testdata/test.mkv` file. This MKV file is specifically designed for the unit tests of `matroska-go` and includes multiple features.

## Generating the Test Dataset

To generate the `test.mkv` file used in the tests, you first need to prepare a source video file and a few sidecar files for features like chapters, subtitles, and attachments.

### Prerequisites

Ensure you have `mkvtoolnix` installed, which provides the `mkvmerge` command-line tool. You can download it from the official website: [mkvtoolnix.download](https://mkvtoolnix.download/).

You will also need a source video file. The test file was originally created from a file named `Countdown.2025.S01E02.Dead.Lots.of.Times.1080p.AMZN.WEB-DL.DDP5.1.H.264-FLUX.mkv`, but any modern MKV file with at least one video and one audio track should work. For simplicity, let's assume you have a source file named `source.mkv`.

### Required Files

You need to create the following files in the `testdata/` directory:

1.  **`testdata/chapters.txt`**: A text file defining the chapter timestamps.
    ```
    CHAPTER01=00:00:00.000
    CHAPTER01NAME=Chapter 1
    CHAPTER02=00:00:10.000
    CHAPTER02NAME=Chapter 2
    ```

2.  **`testdata/test.srt`**: A simple subtitle file.
    ```
    1
    00:00:01,000 --> 00:00:04,000
    This is a test subtitle.

    2
    00:00:05,000 --> 00:00:09,000
    This is another line.
    ```

3.  **`testdata/cover.jpg`**: An image file to be used as an attachment. Any small JPEG file will do.

### Build Command

Once the prerequisite files are in place, you can run the following `bash` script to generate the `test.mkv` file. This command combines the source video with the chapter, subtitle, and attachment files, and sets various metadata fields.

```bash
#!/bin/bash

# Define the source MKV file.
# Replace this with the path to your actual source video file.
SOURCE_MKV="/path/to/your/source.mkv"

# Check if the source file exists
if [ ! -f "$SOURCE_MKV" ]; then
    echo "Error: Source MKV file not found at '$SOURCE_MKV'"
    echo "Please update the SOURCE_MKV variable in this script."
    exit 1
fi

# Create dummy files for testing if they don't exist
# In a real scenario, these would be properly prepared
if [ ! -f "testdata/chapters.txt" ]; then
    echo "Creating dummy chapters.txt..."
    echo "CHAPTER01=00:00:00.000" > testdata/chapters.txt
    echo "CHAPTER01NAME=Chapter 1" >> testdata/chapters.txt
    echo "CHAPTER02=00:00:10.000" >> testdata/chapters.txt
    echo "CHAPTER02NAME=Chapter 2" >> testdata/chapters.txt
fi

if [ ! -f "testdata/test.srt" ]; then
    echo "Creating dummy test.srt..."
    echo "1" > testdata/test.srt
    echo "00:00:01,000 --> 00:00:04,000" >> testdata/test.srt
    echo "This is a test subtitle." >> testdata/test.srt
fi

if [ ! -f "testdata/cover.jpg" ]; then
    echo "Downloading dummy cover.jpg..."
    wget "https://gravatar.com/avatar/dda4eabe6b5ad1e473e201f981b9607a" -O testdata/cover.jpg
fi


# Run the mkvmerge command
mkvmerge \
  -o testdata/test.mkv \
  --title "Comprehensive Test MKV" \
  --chapters testdata/chapters.txt \
  --track-name 0:"Test Video" \
  --track-name 1:"Test Audio" \
  --language 2:eng --track-name 2:"Test Subtitle" testdata/test.srt \
  --attachment-name cover.jpg --attachment-mime-type image/jpeg --attach-file testdata/cover.jpg \
  "$SOURCE_MKV"

echo "Successfully created testdata/test.mkv"```

### Command Explanation

*   `mkvmerge`: The merging tool from the Matroska toolset.
*   `-o testdata/test.mkv`: Specifies the output filename as `testdata/test.mkv`.
*   `--title "Comprehensive Test MKV"`: Sets the global "title" tag for the file.
*   `--chapters testdata/chapters.txt`: Adds chapters defined in the `testdata/chapters.txt` file.
*   `--track-name 0:"Test Video"`: Sets the name of the first track (usually video) to "Test Video".
*   `--track-name 1:"Test Audio"`: Sets the name of the second track (usually audio) to "Test Audio".
*   `--language 2:eng --track-name 2:"Test Subtitle" testdata/test.srt`: Adds an SRT subtitle file. `--language 2:eng` sets its language to English, and `--track-name` names it.
*   `--attachment-name cover.jpg --attachment-mime-type image/jpeg --attach-file testdata/cover.jpg`: Adds an attachment. `--attachment-name` is the name of the attachment in the file, `--attachment-mime-type` is its MIME type, and `--attach-file` specifies the path to the attachment file.
*   `"$SOURCE_MKV"`: This is the input source MKV file.

**Note on Lacing**: By default, `mkvmerge` automatically enables lacing for compatible tracks (e.g., Vorbis audio). There is no direct command-line switch to force it for all track types, as it depends on codec and container compatibility. The default behavior is usually sufficient for testing purposes.