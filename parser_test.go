package matroska

import (
	"bytes"
	"os"
	"testing"
	"time"
)

const testFile = "testdata/test.mkv"

// TestNewMatroskaParser tests the creation of a new parser.
// This test requires a real Matroska file.
func TestNewMatroskaParser(t *testing.T) {
	file, err := os.Open(testFile)
	if err != nil {
		t.Skipf("Skipping test: could not open test file %s: %v", testFile, err)
	}
	defer func() {
		_ = file.Close()
	}()

	parser, err := NewMatroskaParser(file, false)
	if err != nil {
		t.Fatalf("NewMatroskaParser() failed: %v", err)
	}

	if parser.header == nil {
		t.Error("Expected parser to have a non-nil header")
	}
	if parser.segment == nil {
		t.Error("Expected parser to have a non-nil segment")
	}
	if parser.fileInfo == nil {
		t.Error("Expected parser to have non-nil fileInfo")
	}
	if len(parser.tracks) == 0 {
		t.Error("Expected parser to have found some tracks")
	}
}

// TestParseSegmentInfo tests the parsing of the SegmentInfo element.
func TestParseSegmentInfo(t *testing.T) {
	// Create a mock SegmentInfo element
	buf := new(bytes.Buffer)
	// Title
	buf.Write([]byte{0x7B, 0xA9, 0x85, 't', 'i', 't', 'l', 'e'})
	// MuxingApp
	buf.Write([]byte{0x4D, 0x80, 0x84, 't', 'e', 's', 't'})
	// WritingApp
	buf.Write([]byte{0x57, 0x41, 0x8B, 'g', 'o', '-', 'm', 'a', 't', 'r', 'o', 's', 'k', 'a'})
	// TimestampScale
	buf.Write([]byte{0x2A, 0xD7, 0xB1, 0x83, 0x0F, 0x42, 0x40}) // 1,000,000
	// Duration
	buf.Write([]byte{0x44, 0x89, 0x88, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0x86, 0xA0}) // 100000

	parser := &MatroskaParser{
		reader: NewEBMLReader(bytes.NewReader(buf.Bytes())),
	}

	err := parser.parseSegmentInfo(uint64(buf.Len()))
	if err != nil {
		t.Fatalf("parseSegmentInfo() failed: %v", err)
	}

	if parser.fileInfo.Title != "title" {
		t.Errorf("Expected Title 'title', got %q", parser.fileInfo.Title)
	}
	if parser.fileInfo.MuxingApp != "test" {
		t.Errorf("Expected MuxingApp 'test', got %q", parser.fileInfo.MuxingApp)
	}
	if parser.fileInfo.WritingApp != "go-matroska" {
		t.Errorf("Expected WritingApp 'go-matroska', got %q", parser.fileInfo.WritingApp)
	}
	if parser.fileInfo.TimecodeScale != 1000000 {
		t.Errorf("Expected TimecodeScale 1000000, got %d", parser.fileInfo.TimecodeScale)
	}
	if parser.fileInfo.Duration != 100000 {
		t.Errorf("Expected Duration 100000, got %d", parser.fileInfo.Duration)
	}
}

// TestParseTracks tests parsing of the Tracks element.
func TestParseTracks(t *testing.T) {
	// Create a mock Tracks element containing one video and one audio track entry
	trackEntryVideo, _ := createMockTrackEntry(1, TypeVideo, "V_MPEG4/ISO/AVC", "Video", "und")
	trackEntryAudio, _ := createMockTrackEntry(2, TypeAudio, "A_AAC", "Audio", "eng")

	tracksElement := new(bytes.Buffer)
	// Write TrackEntry 1
	tracksElement.Write([]byte{0xAE})
	tracksElement.Write(vintEncode(uint64(len(trackEntryVideo))))
	tracksElement.Write(trackEntryVideo)
	// Write TrackEntry 2
	tracksElement.Write([]byte{0xAE})
	tracksElement.Write(vintEncode(uint64(len(trackEntryAudio))))
	tracksElement.Write(trackEntryAudio)

	parser := &MatroskaParser{
		reader:   NewEBMLReader(bytes.NewReader(tracksElement.Bytes())),
		fileInfo: &SegmentInfo{TimecodeScale: 1000000},
	}

	err := parser.parseTracks(uint64(tracksElement.Len()))
	if err != nil {
		t.Fatalf("parseTracks() failed: %v", err)
	}

	if len(parser.tracks) != 2 {
		t.Fatalf("Expected 2 tracks, got %d", len(parser.tracks))
	}

	// Video track checks
	videoTrack := parser.tracks[0]
	if videoTrack.Number != 1 {
		t.Errorf("Expected video track number 1, got %d", videoTrack.Number)
	}
	if videoTrack.Type != TypeVideo {
		t.Errorf("Expected video track type %d, got %d", TypeVideo, videoTrack.Type)
	}
	if videoTrack.CodecID != "V_MPEG4/ISO/AVC" {
		t.Errorf("Expected video CodecID 'V_MPEG4/ISO/AVC', got %q", videoTrack.CodecID)
	}
	if videoTrack.Name != "Video" {
		t.Errorf("Expected video name 'Video', got %q", videoTrack.Name)
	}

	// Audio track checks
	audioTrack := parser.tracks[1]
	if audioTrack.Number != 2 {
		t.Errorf("Expected audio track number 2, got %d", audioTrack.Number)
	}
	if audioTrack.Type != TypeAudio {
		t.Errorf("Expected audio track type %d, got %d", TypeAudio, audioTrack.Type)
	}
	if audioTrack.CodecID != "A_AAC" {
		t.Errorf("Expected audio CodecID 'A_AAC', got %q", audioTrack.CodecID)
	}
	if audioTrack.Language != "eng" {
		t.Errorf("Expected audio language 'eng', got %q", audioTrack.Language)
	}
}

// TestParseSimpleBlock tests the parsing of a SimpleBlock.
func TestParseSimpleBlock(t *testing.T) {
	// SimpleBlock: Track 1, Timecode 1234, Flags 0x80 (Keyframe), Data "frame"
	blockData := []byte{
		0x81,       // Track number 1
		0x04, 0xD2, // Timecode 1234
		0x80,                    // Flags (keyframe)
		'f', 'r', 'a', 'm', 'e', // Frame data
	}

	parser := &MatroskaParser{
		reader:           NewEBMLReader(bytes.NewReader(blockData)),
		clusterTimestamp: 1000,
		fileInfo: &SegmentInfo{
			TimecodeScale: uint64(time.Second / time.Nanosecond), // 1ms
		},
	}

	packet, err := parser.parseSimpleBlock(uint64(len(blockData)))
	if err != nil {
		t.Fatalf("parseSimpleBlock() failed: %v", err)
	}

	if packet.Track != 1 {
		t.Errorf("Expected track 1, got %d", packet.Track)
	}
	expectedTime := (1000 + 1234) * (uint64(time.Second) / uint64(time.Nanosecond))
	if packet.StartTime != expectedTime {
		t.Errorf("Expected start time %d, got %d", expectedTime, packet.StartTime)
	}
	if (packet.Flags & KF) == 0 {
		t.Error("Expected keyframe flag to be set")
	}
	if string(packet.Data) != "frame" {
		t.Errorf("Expected data 'frame', got %q", string(packet.Data))
	}
}

// Helper to create a mock TrackEntry element
func createMockTrackEntry(number uint8, trackType uint8, codecID, name, language string) ([]byte, error) {
	buf := new(bytes.Buffer)

	// TrackNumber
	buf.Write([]byte{0xD7, 0x81, byte(number)})
	// TrackType
	buf.Write([]byte{0x83, 0x81, byte(trackType)})
	// CodecID
	buf.Write([]byte{0x86, byte(0x80 | len(codecID))})
	buf.WriteString(codecID)
	// Name
	buf.Write([]byte{0x53, 0x6E, byte(0x80 | len(name))})
	buf.WriteString(name)
	// Language
	buf.Write([]byte{0x22, 0xB5, 0x9C, byte(0x80 | len(language))})
	buf.WriteString(language)

	return buf.Bytes(), nil
}

// Helper to encode a uint64 into a VINT
func vintEncode(val uint64) []byte {
	if val < (1<<7)-1 {
		return []byte{byte(val | 1<<7)}
	}
	if val < (1<<14)-1 {
		return []byte{byte(val>>8 | 1<<6), byte(val)}
	}
	if val < (1<<21)-1 {
		return []byte{byte(val>>16 | 1<<5), byte(val >> 8), byte(val)}
	}
	if val < (1<<28)-1 {
		return []byte{byte(val>>24 | 1<<4), byte(val >> 16), byte(val >> 8), byte(val)}
	}
	// Add more cases if needed
	return nil
}
