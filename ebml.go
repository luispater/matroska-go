// Package matroska provides functionality for parsing Matroska/EBML (Extensible Binary Meta Language) files.
//
// Matroska is a multimedia container format that can hold an unlimited number of video, audio,
// picture, or subtitle tracks in one file. It is based on EBML, which is a binary format similar to XML.
//
// This package implements the core EBML parsing functionality, including:
//   - Reading and parsing EBML elements
//   - Handling variable-length integers (VINT)
//   - Extracting different data types from elements
//   - Reading and parsing the EBML header
//
// The main types in this package are:
//   - EBMLElement: Represents a single EBML element with ID, size, and data
//   - EBMLReader: Provides methods for reading EBML data from a stream
//   - EBMLHeader: Represents the EBML header containing metadata about the file
//
// Example usage:
//
//	file, err := os.Open("video.mkv")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer file.Close()
//
//	reader := matroska.NewEBMLReader(file)
//	header, err := reader.ReadEBMLHeader()
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	fmt.Printf("DocType: %s, Version: %d\n", header.DocType, header.DocTypeVersion)
package matroska

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"math"
)

// EBML element IDs for Matroska
//
// These constants define the standard element IDs used in Matroska/EBML files.
// Each ID is a unique identifier for a specific element type in the EBML structure.
const (
	// EBML Header elements
	IDEBMLHeader             = 0x1A45DFA3 // The EBML header element
	IDEBMLVersion            = 0x4286     // The version of EBML parser used to create the file
	IDEBMLReadVersion        = 0x42F7     // The minimum EBML version needed to parse this file
	IDEBMLMaxIDLength        = 0x42F2     // The maximum length of an EBML ID in bytes
	IDEBMLMaxSizeLength      = 0x42F3     // The maximum length of an EBML size in bytes
	IDEBMLDocType            = 0x4282     // A string that describes the type of document (e.g., "matroska")
	IDEBMLDocTypeVersion     = 0x4287     // The version of the document type
	IDEBMLDocTypeReadVersion = 0x4285     // The minimum version of the document type parser needed to read this file

	// Segment elements
	IDSegment = 0x18538067 // The root element that contains all other top-level elements

	// Meta Seek Information elements
	IDSeekHead = 0x114D9B74 // Contains a list of seek points to other EBML elements
	IDSeek     = 0x4DBB     // A single seek point to an EBML element
	IDSeekID   = 0x53AB     // The ID of the element to seek to
	IDSeekPos  = 0x53AC     // The position of the element in the segment

	// Segment Information elements
	IDSegmentInfo      = 0x1549A966 // Contains general information about the segment
	IDSegmentUID       = 0x73A4     // A unique identifier for the segment
	IDSegmentFilename  = 0x7384     // The filename corresponding to this segment
	IDPrevUID          = 0x3CB923   // The UID of the previous segment
	IDPrevFilename     = 0x3C83AB   // The filename of the previous segment
	IDNextUID          = 0x3EB923   // The UID of the next segment
	IDNextFilename     = 0x3E83BB   // The filename of the next segment
	IDSegmentFamily    = 0x4444     // A family of segments this segment belongs to
	IDChapterTranslate = 0x6924     // Contains information for translating chapter numbers
	IDTimestampScale   = 0x2AD7B1   // The scale factor for all timestamps in the segment
	IDDuration         = 0x4489     // The duration of the segment in timestamp units
	IDDateUTC          = 0x4461     // The date and time the segment was created in UTC
	IDTitle            = 0x7BA9     // The title of the segment
	IDMuxingApp        = 0x4D80     // The name of the application used to mux the file
	IDWritingApp       = 0x5741     // The name of the application used to write the file

	// Track elements
	IDTracks     = 0x1654AE6B // A top-level element containing all track entries
	IDTrackEntry = 0xAE       // A single track entry containing information about a track
	IDTrackNum   = 0xD7       // The track number as used in the Block header
	IDTrackUID   = 0x73C5     // A unique identifier for the track
	IDTrackType  = 0x83       // The type of the track (video, audio, etc.)
	IDTrackName  = 0x536E     // The name of the track
	IDLanguage   = 0x22B59C   // The language of the track
	IDCodecID    = 0x86       // The ID of the codec used for this track
	IDCodecPriv  = 0x63A2     // Private data specific to the codec
	IDCodecName  = 0x258688   // The name of the codec used for this track
	IDVideo      = 0xE0       // Video settings specific to this track
	IDAudio      = 0xE1       // Audio settings specific to this track

	// Video elements
	IDFlagInterlaced = 0x9A   // Flag indicating whether the video is interlaced
	IDPixelWidth     = 0xB0   // The width of the encoded video frames in pixels
	IDPixelHeight    = 0xBA   // The height of the encoded video frames in pixels
	IDDisplayWidth   = 0x54B0 // The width of the video frames when displayed
	IDDisplayHeight  = 0x54BA // The height of the video frames when displayed

	// Audio elements
	IDSamplingFrequency       = 0xB5   // The sampling frequency of the audio in Hz
	IDOutputSamplingFrequency = 0x78B5 // The output sampling frequency of the audio in Hz
	IDChannels                = 0x9F   // The number of audio channels
	IDBitDepth                = 0x6264 // The number of bits per audio sample

	// Cluster elements
	IDCluster     = 0x1F43B675 // A cluster contains blocks of data for a specific timestamp
	IDTimestamp   = 0xE7       // The timestamp of the cluster
	IDSimpleBlock = 0xA3       // A block containing raw data without additional metadata
	IDBlockGroup  = 0xA0       // A group of blocks with additional metadata
	IDBlock       = 0xA1       // A block containing raw data

	// Cues elements
	IDCues     = 0x1C53BB6B // A top-level element containing all cue points
	IDCuePoint = 0xBB       // A single cue point pointing to a specific timestamp
	IDCueTime  = 0xB3       // The timestamp of the cue point

	// Chapters elements
	IDChapters = 0x1043A770 // A top-level element containing all chapter entries

	// Tags elements
	IDTags = 0x1254C367 // A top-level element containing all tags

	// Attachments elements
	IDAttachments = 0x1941A469 // A top-level element containing all attached files
)

// EBMLElement represents an EBML element with its ID, size, and data.
//
// An EBML element is the basic building block of EBML files. Each element consists of:
//   - ID: A variable-length integer that identifies the type of element
//   - Size: A variable-length integer that specifies the size of the element's data
//   - Data: The actual data contained within the element
//
// The EBMLElement struct provides methods to extract different types of data from the element,
// such as integers, floats, strings, and raw bytes.
type EBMLElement struct {
	ID   uint32 // The element ID that identifies the type of element
	Size uint64 // The size of the element's data in bytes
	Data []byte // The raw data contained within the element
}

// EBMLReader provides methods for reading EBML data from a stream.
//
// EBMLReader is the main type used for parsing EBML data. It wraps an io.ReadSeeker
// and provides methods to read EBML elements, variable-length integers, and other
// EBML-specific data structures.
//
// Example usage:
//
//	file, err := os.Open("video.mkv")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer file.Close()
//
//	reader := NewEBMLReader(file)
//	element, err := reader.ReadElement()
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	fmt.Printf("Element ID: 0x%X, Size: %d\n", element.ID, element.Size)
type EBMLReader struct {
	r   io.ReadSeeker // The underlying reader for the EBML data
	pos int64         // The current position in the stream
}

// NewEBMLReader creates a new EBML reader from an io.ReadSeeker.
//
// This function initializes a new EBMLReader with the provided io.ReadSeeker.
// The reader is used to read EBML data from a stream, such as a file or network connection.
//
// Parameters:
//   - r: An io.ReadSeeker that provides the EBML data stream
//
// Returns:
//   - A pointer to the newly created EBMLReader
//
// Example usage:
//
//	file, err := os.Open("video.mkv")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer file.Close()
//
//	reader := NewEBMLReader(file)
func NewEBMLReader(r io.ReadSeeker) *EBMLReader {
	return &EBMLReader{r: r}
}

// ReadVInt reads a variable-length integer from the stream.
//
// Variable-length integers (VINT) are used in EBML to store element sizes and other values.
// This method reads a VINT and removes the length marker, returning only the value.
//
// Returns:
//   - The value of the variable-length integer
//   - An error if the read operation failed or the VINT is invalid
func (er *EBMLReader) ReadVInt() (uint64, error) {
	return er.readVInt(false)
}

// ReadVIntID reads a variable-length integer for element IDs, keeping the length marker.
//
// This method is similar to ReadVInt, but it preserves the length marker in the returned value.
// It is used specifically for reading EBML element IDs, which require the length marker to be preserved.
//
// Returns:
//   - The value of the variable-length integer including the length marker
//   - An error if the read operation failed or the VINT is invalid
func (er *EBMLReader) ReadVIntID() (uint64, error) {
	return er.readVInt(true)
}

// readVInt reads a variable-length integer from the stream.
//
// This is the internal implementation for reading variable-length integers (VINT).
// A VINT consists of a length marker in the first byte followed by the actual value.
// The length marker indicates how many bytes are used to store the value.
//
// Parameters:
//   - keepLengthMarker: If true, the length marker is included in the returned value.
//     If false, only the value part is returned.
//
// Returns:
//   - The value of the variable-length integer
//   - An error if the read operation failed or the VINT is invalid
func (er *EBMLReader) readVInt(keepLengthMarker bool) (uint64, error) {
	var b [1]byte
	if _, err := er.r.Read(b[:]); err != nil {
		return 0, err
	}

	er.pos++

	// Find the number of bytes to read based on the first bit pattern
	firstByte := b[0]
	if firstByte == 0 {
		return 0, fmt.Errorf("invalid VINT: first byte is 0")
	}

	// Count leading zeros to determine length
	var length int
	var lengthMask uint8

	if firstByte&0x80 != 0 {
		length = 1
		lengthMask = 0x80
	} else if firstByte&0x40 != 0 {
		length = 2
		lengthMask = 0x40
	} else if firstByte&0x20 != 0 {
		length = 3
		lengthMask = 0x20
	} else if firstByte&0x10 != 0 {
		length = 4
		lengthMask = 0x10
	} else if firstByte&0x08 != 0 {
		length = 5
		lengthMask = 0x08
	} else if firstByte&0x04 != 0 {
		length = 6
		lengthMask = 0x04
	} else if firstByte&0x02 != 0 {
		length = 7
		lengthMask = 0x02
	} else if firstByte&0x01 != 0 {
		length = 8
		lengthMask = 0x01
	} else {
		return 0, fmt.Errorf("invalid VINT: no length marker found")
	}

	// Start with the first byte
	var result uint64
	if keepLengthMarker {
		result = uint64(firstByte)
	} else {
		result = uint64(firstByte & (lengthMask - 1))
	}

	// Read remaining bytes
	for i := 1; i < length; i++ {
		if _, err := er.r.Read(b[:]); err != nil {
			return 0, err
		}
		er.pos++
		result = (result << 8) | uint64(b[0])
	}

	return result, nil
}

// ReadElement reads a complete EBML element from the stream.
//
// This method reads an EBML element, which consists of an ID, a size, and the element data.
// It first reads the element ID using ReadVIntID, then reads the element size using ReadVInt,
// and finally reads the element data based on the size.
//
// Returns:
//   - A pointer to the EBMLElement that was read
//   - An error if the read operation failed or the element is invalid
//
// Example usage:
//
//	element, err := reader.ReadElement()
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	fmt.Printf("Element ID: 0x%X, Size: %d\n", element.ID, element.Size)
func (er *EBMLReader) ReadElement() (*EBMLElement, error) {
	// Read element ID (keep length marker for IDs)
	id, err := er.ReadVIntID()
	if err != nil {
		return nil, fmt.Errorf("failed to read element ID: %w", err)
	}

	// Read element size (remove length marker for sizes)
	size, err := er.ReadVInt()
	if err != nil {
		return nil, fmt.Errorf("failed to read element size: %w", err)
	}

	// Check for unknown size marker
	if size == (1<<(7*8))-1 {
		return nil, fmt.Errorf("unknown size elements not supported")
	}

	// Read element data
	data := make([]byte, size)
	if size > 0 {
		n, errReadFull := io.ReadFull(er.r, data)
		if errReadFull != nil {
			return nil, fmt.Errorf("failed to read element data: %w", errReadFull)
		}
		er.pos += int64(n)
	}

	return &EBMLElement{
		ID:   uint32(id),
		Size: size,
		Data: data,
	}, nil
}

// Seek moves the reader to the specified position in the stream.
//
// This method implements the io.Seeker interface, allowing random access to the EBML data.
// It delegates to the underlying io.ReadSeeker and updates the internal position tracker.
//
// Parameters:
//   - offset: The offset to seek to, relative to the whence parameter
//   - whence: The reference point for the offset (0 = beginning, 1 = current, 2 = end)
//
// Returns:
//   - The new position relative to the beginning of the stream
//   - An error if the seek operation failed
func (er *EBMLReader) Seek(offset int64, whence int) (int64, error) {
	pos, err := er.r.Seek(offset, whence)
	if err != nil {
		return 0, err
	}
	er.pos = pos
	return pos, nil
}

// Position returns the current position in the stream.
//
// This method returns the current position of the reader in the stream,
// which is tracked internally and updated after each read or seek operation.
//
// Returns:
//   - The current position in the stream as a byte offset from the beginning
func (er *EBMLReader) Position() int64 {
	return er.pos
}

// ReadUInt reads an unsigned integer from the element's data.
//
// This method interprets the element's data as a big-endian unsigned integer
// and returns its value. If the element's data is empty, it returns 0.
//
// Returns:
//   - The unsigned integer value stored in the element's data
//
// Example usage:
//
//	value := element.ReadUInt()
//	fmt.Printf("Value: %d\n", value)
func (el *EBMLElement) ReadUInt() uint64 {
	if len(el.Data) == 0 {
		return 0
	}

	var result uint64
	for _, b := range el.Data {
		result = (result << 8) | uint64(b)
	}
	return result
}

// ReadInt reads a signed integer from element data
func (el *EBMLElement) ReadInt() int64 {
	if len(el.Data) == 0 {
		return 0
	}

	// Check sign bit
	isNegative := el.Data[0]&0x80 != 0

	var result uint64
	for _, b := range el.Data {
		result = (result << 8) | uint64(b)
	}

	if isNegative {
		// Two's complement for negative numbers
		switch len(el.Data) {
		case 1:
			return int64(int8(result))
		case 2:
			return int64(int16(result))
		case 4:
			return int64(int32(result))
		case 8:
			return int64(result)
		default:
			// Handle arbitrary length negative numbers
			mask := uint64(1<<(uint(len(el.Data))*8-1)) - 1
			return -int64((^result & mask) + 1)
		}
	}

	return int64(result)
}

// ReadFloat reads a floating-point number from the element's data.
//
// This method interprets the element's data as a big-endian floating-point number
// (either 32-bit or 64-bit) and returns its value. If the element's data is empty
// or its length is not 4 or 8 bytes, it returns 0.0.
//
// Returns:
//   - The floating-point value stored in the element's data.
func (el *EBMLElement) ReadFloat() float64 {
	if len(el.Data) == 0 {
		return 0.0
	}

	switch len(el.Data) {
	case 4:
		bits := binary.BigEndian.Uint32(el.Data)
		return float64(math.Float32frombits(bits))
	case 8:
		bits := binary.BigEndian.Uint64(el.Data)
		return math.Float64frombits(bits)
	default:
		return 0.0
	}
}

// ReadString reads a UTF-8 string from the element's data.
//
// This method interprets the element's data as a UTF-8 encoded string.
// It removes any null terminator if present at the end of the data.
//
// Returns:
//   - The string value stored in the element's data.
func (el *EBMLElement) ReadString() string {
	// Remove null terminator if present
	data := el.Data
	if len(data) > 0 && data[len(data)-1] == 0 {
		data = data[:len(data)-1]
	}
	return string(data)
}

// ReadBytes returns the raw byte slice of the element's data.
//
// This method provides direct access to the uninterpreted byte data
// contained within the EBML element.
//
// Returns:
//   - A byte slice containing the raw data of the element.
func (el *EBMLElement) ReadBytes() []byte {
	return el.Data
}

// SkipElement skips the current element by seeking past its data in the stream.
//
// This method is useful for efficiently moving past elements whose content
// is not needed for current processing. It updates the reader's internal
// position tracker.
//
// Parameters:
//   - element: The EBMLElement to skip.
//
// Returns:
//   - An error if the seek operation failed.
func (er *EBMLReader) SkipElement(element *EBMLElement) error {
	_, err := er.r.Seek(int64(element.Size), io.SeekCurrent)
	if err != nil {
		return err
	}
	er.pos += int64(element.Size)
	return nil
}

// ReadElementHeader reads only the element ID and size from the stream, without reading the actual data.
//
// This method is useful when you only need to inspect the type and size of an element
// before deciding whether to read its full content or skip it.
//
// Returns:
//   - The ID of the element.
//   - The size of the element's data.
//   - An error if the read operation failed.
func (er *EBMLReader) ReadElementHeader() (uint32, uint64, error) {
	// Read element ID (keep length marker for IDs)
	id, err := er.ReadVIntID()
	if err != nil {
		return 0, 0, fmt.Errorf("failed to read element ID: %w", err)
	}

	// Read element size (remove length marker for sizes)
	size, err := er.ReadVInt()
	if err != nil {
		return 0, 0, fmt.Errorf("failed to read element size: %w", err)
	}

	return uint32(id), size, nil
}

// EBMLHeader represents the EBML header containing metadata about the file.
//
// The EBML header is the first element in an EBML file and contains information
// about how to parse the rest of the file. It includes the EBML version, document type,
// and other metadata that helps parsers understand the structure of the file.
//
// Example usage:
//
//	reader := NewEBMLReader(file)
//	header, err := reader.ReadEBMLHeader()
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	fmt.Printf("DocType: %s, Version: %d\n", header.DocType, header.DocTypeVersion)
type EBMLHeader struct {
	Version            uint64 // The version of EBML parser used to create the file
	ReadVersion        uint64 // The minimum EBML version needed to parse this file
	MaxIDLength        uint64 // The maximum length of an EBML ID in bytes
	MaxSizeLength      uint64 // The maximum length of an EBML size in bytes
	DocType            string // A string that describes the type of document (e.g., "matroska")
	DocTypeVersion     uint64 // The version of the document type
	DocTypeReadVersion uint64 // The minimum version of the document type parser needed to read this file
}

// ReadEBMLHeader reads and parses the EBML header from the stream.
//
// This method expects the next element in the stream to be the EBML header (IDEBMLHeader).
// It reads the header element and then parses its child elements to populate the
// EBMLHeader struct.
//
// Returns:
//   - A pointer to the parsed EBMLHeader.
//   - An error if reading the header fails or if the first element is not an EBML header.
func (er *EBMLReader) ReadEBMLHeader() (*EBMLHeader, error) {
	// Read EBML header element
	element, err := er.ReadElement()
	if err != nil {
		return nil, fmt.Errorf("failed to read EBML header: %w", err)
	}

	if element.ID != IDEBMLHeader {
		return nil, fmt.Errorf("expected EBML header, got ID 0x%X", element.ID)
	}

	header := &EBMLHeader{}
	reader := bytes.NewReader(element.Data)
	childReader := &EBMLReader{r: &seekableReader{reader}, pos: 0}

	for childReader.pos < int64(len(element.Data)) {
		childElement, errReadElement := childReader.ReadElement()
		if errReadElement != nil {
			if errReadElement == io.EOF {
				break
			}
			return nil, fmt.Errorf("failed to read header child element: %w", errReadElement)
		}

		switch childElement.ID {
		case IDEBMLVersion:
			header.Version = childElement.ReadUInt()
		case IDEBMLReadVersion:
			header.ReadVersion = childElement.ReadUInt()
		case IDEBMLMaxIDLength:
			header.MaxIDLength = childElement.ReadUInt()
		case IDEBMLMaxSizeLength:
			header.MaxSizeLength = childElement.ReadUInt()
		case IDEBMLDocType:
			header.DocType = childElement.ReadString()
		case IDEBMLDocTypeVersion:
			header.DocTypeVersion = childElement.ReadUInt()
		case IDEBMLDocTypeReadVersion:
			header.DocTypeReadVersion = childElement.ReadUInt()
		}
	}

	return header, nil
}

// seekableReader wraps a bytes.Reader to implement io.ReadSeeker.
//
// This is a helper type that allows a bytes.Reader to be used as an io.ReadSeeker,
// which is required by the EBMLReader. It simply delegates all operations to the
// underlying bytes.Reader.
type seekableReader struct {
	*bytes.Reader // The underlying bytes.Reader
}

// Seek implements the io.Seeker interface for seekableReader.
//
// It delegates the Seek operation to the underlying bytes.Reader.
//
// Parameters:
//   - offset: The offset to seek to, relative to the whence parameter
//   - whence: The reference point for the offset (0 = beginning, 1 = current, 2 = end)
//
// Returns:
//   - The new position relative to the beginning of the stream
//   - An error if the seek operation failed
func (sr *seekableReader) Seek(offset int64, whence int) (int64, error) {
	return sr.Reader.Seek(offset, whence)
}
