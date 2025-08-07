package matroska

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"testing"
)

const testDemuxerFile = "testdata/test.mkv"

// TestDemuxer tests the high-level Demuxer API with a real file.
func TestDemuxer(t *testing.T) {
	file, err := os.Open(testDemuxerFile)
	if err != nil {
		t.Skipf("Skipping demuxer test: could not open test file %s: %v", testDemuxerFile, err)
	}
	defer func() {
		_ = file.Close()
	}()

	demuxer, err := NewDemuxer(file)
	if err != nil {
		t.Fatalf("NewDemuxer() failed: %v", err)
	}
	defer demuxer.Close()

	// Test GetFileInfo
	fileInfo, err := demuxer.GetFileInfo()
	if err != nil {
		t.Fatalf("GetFileInfo() failed: %v", err)
	}
	if fileInfo == nil {
		t.Fatal("GetFileInfo() returned nil info")
	}
	if fileInfo.Title == "" {
		t.Log("Warning: File info title is empty")
	}

	// Test GetNumTracks and GetTrackInfo
	numTracks, err := demuxer.GetNumTracks()
	if err != nil {
		t.Fatalf("GetNumTracks() failed: %v", err)
	}
	if numTracks == 0 {
		t.Fatal("Expected at least one track")
	}

	for i := uint(0); i < numTracks; i++ {
		trackInfo, errGetTrackInfo := demuxer.GetTrackInfo(i)
		if errGetTrackInfo != nil {
			t.Fatalf("GetTrackInfo(%d) failed: %v", i, errGetTrackInfo)
		}
		if trackInfo == nil {
			t.Fatalf("GetTrackInfo(%d) returned nil info", i)
		}
	}

	// Test ReadPacket
	// Read a few packets to ensure it works
	for i := 0; i < 5; i++ {
		packet, errReadPacket := demuxer.ReadPacket()
		if errReadPacket != nil {
			if errReadPacket == io.EOF {
				t.Log("Reached EOF after reading packets")
				break
			}
			t.Fatalf("ReadPacket() failed after %d packets: %v", i, errReadPacket)
		}
		if packet == nil {
			t.Fatal("ReadPacket() returned nil packet")
		}
	}
}

// nonSeekableReader wraps an io.Reader to make it non-seekable for tests.
type nonSeekableReader struct {
	r io.Reader
}

func (r *nonSeekableReader) Read(p []byte) (n int, err error) {
	return r.r.Read(p)
}

func (r *nonSeekableReader) Seek(offset int64, whence int) (int64, error) {
	return -1, fmt.Errorf("this is a fake seeker")
}

// TestStreamingDemuxer tests the Demuxer with a non-seekable stream.
func TestStreamingDemuxer(t *testing.T) {
	// We can't use a real file here directly because it needs to be non-seekable.
	// We will create a mock in-memory Matroska file for testing.
	mockFile, err := createMockMatroskaFile()
	if err != nil {
		t.Fatalf("Failed to create mock matroska file: %v", err)
	}

	reader := &nonSeekableReader{r: bytes.NewReader(mockFile)}

	demuxer, err := NewStreamingDemuxer(reader)
	if err != nil {
		t.Fatalf("NewStreamingDemuxer() failed: %v", err)
	}
	defer demuxer.Close()

	// Test GetFileInfo
	fileInfo, err := demuxer.GetFileInfo()
	if err != nil {
		t.Fatalf("GetFileInfo() failed: %v", err)
	}
	if fileInfo.Title != "Test Title" {
		t.Errorf("Expected title 'Test Title', got %q", fileInfo.Title)
	}

	// Test GetNumTracks
	numTracks, err := demuxer.GetNumTracks()
	if err != nil {
		t.Fatalf("GetNumTracks() failed: %v", err)
	}
	if numTracks != 1 {
		t.Fatalf("Expected 1 track, got %d", numTracks)
	}

	// Test ReadPacket
	packet, err := demuxer.ReadPacket()
	if err != nil && err != io.EOF {
		t.Fatalf("ReadPacket() failed: %v", err)
	}
	if packet != nil {
		if packet.Track != 1 {
			t.Errorf("Expected packet for track 1, got %d", packet.Track)
		}
		if string(packet.Data) != "frame" {
			t.Errorf("Expected packet data 'frame', got %q", string(packet.Data))
		}
	} else if err != io.EOF {
		t.Error("Expected to read a packet or get EOF")
	}
}

// createMockMatroskaFile creates a minimal valid Matroska file in memory.
func createMockMatroskaFile() ([]byte, error) {
	buf := new(bytes.Buffer)

	// EBML Header
	ebmlHeader := new(bytes.Buffer)
	ebmlHeader.Write([]byte{0x42, 0x82, 0x88, 'm', 'a', 't', 'r', 'o', 's', 'k', 'a'}) // DocType
	buf.Write([]byte{0x1A, 0x45, 0xDF, 0xA3})                                          // EBML Header ID
	buf.Write(vintEncode(uint64(ebmlHeader.Len())))
	buf.Write(ebmlHeader.Bytes())

	// Segment
	segment := new(bytes.Buffer)

	// -- SegmentInfo
	segInfo := new(bytes.Buffer)
	segInfo.Write([]byte{0x7B, 0xA9, 0x8A, 'T', 'e', 's', 't', ' ', 'T', 'i', 't', 'l', 'e'}) // Title
	segInfo.Write([]byte{0x2A, 0xD7, 0xB1, 0x83, 0x0F, 0x42, 0x40})                           // TimestampScale 1,000,000
	segment.Write([]byte{0x15, 0x49, 0xA9, 0x66})                                             // SegmentInfo ID
	segment.Write(vintEncode(uint64(segInfo.Len())))
	segment.Write(segInfo.Bytes())

	// -- Tracks
	trackEntry, _ := createMockTrackEntry(1, TypeVideo, "V_TEST", "TestVideo", "und")
	tracks := new(bytes.Buffer)
	tracks.Write([]byte{0xAE}) // TrackEntry ID
	tracks.Write(vintEncode(uint64(len(trackEntry))))
	tracks.Write(trackEntry)
	segment.Write([]byte{0x16, 0x54, 0xAE, 0x6B}) // Tracks ID
	segment.Write(vintEncode(uint64(tracks.Len())))
	segment.Write(tracks.Bytes())

	// -- Cluster
	cluster := new(bytes.Buffer)
	cluster.Write([]byte{0xE7, 0x81, 0x00}) // Timestamp 0
	// SimpleBlock: Track 1, Timecode 0, Flags 0x80 (Keyframe), Data "frame"
	blockData := []byte{0x81, 0x00, 0x00, 0x80, 'f', 'r', 'a', 'm', 'e'}
	cluster.Write([]byte{0xA3, byte(0x80 | len(blockData))})
	cluster.Write(blockData)
	segment.Write([]byte{0x1F, 0x43, 0xB6, 0x75}) // Cluster ID
	segment.Write(vintEncode(uint64(cluster.Len())))
	segment.Write(cluster.Bytes())

	buf.Write([]byte{0x18, 0x53, 0x80, 0x67}) // Segment ID
	// Unknown size for streaming
	buf.Write([]byte{0x01, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF})
	buf.Write(segment.Bytes())

	return buf.Bytes(), nil
}
