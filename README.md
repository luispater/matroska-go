# Pure Go Matroska Parser

A high-performance, pure Go implementation of a Matroska (MKV) container parser and demuxer.

## Overview

This Go package provides a complete solution for parsing Matroska video files without CGO dependencies. It offers both high-level demuxing capabilities and low-level EBML parsing, making it suitable for applications ranging from simple media information extraction to full-featured media processing tools.

## Features

- **Pure Go Implementation**: No CGO dependencies, easy cross-compilation
- **Complete EBML/Matroska Support**: Full parsing of EBML structure and Matroska-specific elements
- **Streaming Support**: Works with both seekable files and non-seekable streams
- **Multi-track Demuxing**: Support for video, audio, and subtitle tracks
- **Codec Integration**: Handles codec private data and format conversion (AVCC to Annex B for H.264/H.265)
- **Subtitle Processing**: Built-in SRT format conversion for subtitle tracks
- **Memory Efficient**: Stream-based processing with minimal memory footprint

## Installation

```bash
go get github.com/luispater/matroska-go
```

## Quick Start

### Basic Usage

```go
package main

import (
    "fmt"
    "io"
    "os"
    "github.com/luispater/matroska-go"
)

func main() {
    file, err := os.Open("video.mkv")
    if err != nil {
        panic(err)
    }
    defer file.Close()

    // Create demuxer
    demuxer, err := matroska.NewDemuxer(file)
    if err != nil {
        panic(err)
    }
    defer demuxer.Close()

    // Get file information
    fileInfo, _ := demuxer.GetFileInfo()
    fmt.Printf("Duration: %d ms\n", fileInfo.Duration)

    // Get track information
    numTracks, _ := demuxer.GetNumTracks()
    for i := uint(0); i < numTracks; i++ {
        track, _ := demuxer.GetTrackInfo(i)
        fmt.Printf("Track %d: Type=%d, Codec=%s\n", i, track.Type, track.CodecID)
    }

    // Read packets
    for {
        packet, err := demuxer.ReadPacket()
        if err == io.EOF {
            break
        }
        if err != nil {
            panic(err)
        }
        fmt.Printf("Track %d: %d bytes at %d ms\n", 
            packet.Track, len(packet.Data), packet.StartTime)
    }
}
```

### Streaming Support

For non-seekable streams (e.g., network streams):

```go
// Use NewStreamingDemuxer for io.Reader streams
demuxer, err := matroska.NewStreamingDemuxer(networkStream)
```

## Track Extraction Tool

A complete track extraction tool is included that demonstrates advanced usage:

```bash
go run ./example/extracter/main.go
```

This tool extracts all tracks from MKV files with:
- Video tracks: Converts AVCC format to Annex B format with proper NAL unit handling
- Audio tracks: Raw stream extraction  
- Subtitle tracks: Converts to SRT format with proper timing

## Architecture

The library is structured in three layers:

1. **EBML Layer** (`ebml.go`): Low-level EBML parsing
2. **Matroska Parser** (`parser.go`): Matroska-specific structure parsing
3. **Demuxer API** (`matroska.go`): High-level demuxing interface

## Track Types

- **Video (Type 1)**: H.264, H.265, VP8, VP9, AV1
- **Audio (Type 2)**: AAC, AC-3, DTS, FLAC, Opus, Vorbis
- **Subtitle (Type 17)**: ASS, SRT, VobSub, PGS

## API Reference

### Core Types

```go
type Packet struct {
    Track     uint8   // Track number
    StartTime uint64  // Start time in milliseconds
    EndTime   uint64  // End time in milliseconds
    Data      []byte  // Packet data
    Flags     uint32  // Packet flags (keyframe, etc.)
}

type TrackInfo struct {
    Number       uint8   // Track number
    Type         uint8   // Track type (1=video, 2=audio, 17=subtitle)
    CodecID      string  // Codec identifier
    CodecPrivate []byte  // Codec-specific data
    // ... additional fields
}
```

### Main Functions

- `NewDemuxer(io.ReadSeeker) (*Demuxer, error)` - Create demuxer for seekable streams
- `NewStreamingDemuxer(io.Reader) (*Demuxer, error)` - Create demuxer for streaming
- `GetNumTracks() (uint, error)` - Get number of tracks
- `GetTrackInfo(uint) (*TrackInfo, error)` - Get track information
- `ReadPacket() (*Packet, error)` - Read next packet
- `GetFileInfo() (*SegmentInfo, error)` - Get file metadata

## Requirements

- Go 1.24 or later
- No external dependencies

## Performance

This pure Go implementation achieves 100% accuracy compared to reference implementations while maintaining high performance through:

- Stream-based processing
- Efficient EBML parsing
- Minimal memory allocations
- Optimized codec format conversions

## License

Distributed under the MIT License. See the [LICENSE](LICENSE) file for details.
