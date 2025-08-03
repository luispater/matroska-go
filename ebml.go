package matroska

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"math"
)

// EBML element IDs for Matroska
const (
	IDEBMLHeader             = 0x1A45DFA3
	IDEBMLVersion            = 0x4286
	IDEBMLReadVersion        = 0x42F7
	IDEBMLMaxIDLength        = 0x42F2
	IDEBMLMaxSizeLength      = 0x42F3
	IDEBMLDocType            = 0x4282
	IDEBMLDocTypeVersion     = 0x4287
	IDEBMLDocTypeReadVersion = 0x4285

	// Segment
	IDSegment = 0x18538067

	// Meta Seek Information
	IDSeekHead = 0x114D9B74
	IDSeek     = 0x4DBB
	IDSeekID   = 0x53AB
	IDSeekPos  = 0x53AC

	// Segment Information
	IDSegmentInfo      = 0x1549A966
	IDSegmentUID       = 0x73A4
	IDSegmentFilename  = 0x7384
	IDPrevUID          = 0x3CB923
	IDPrevFilename     = 0x3C83AB
	IDNextUID          = 0x3EB923
	IDNextFilename     = 0x3E83BB
	IDSegmentFamily    = 0x4444
	IDChapterTranslate = 0x6924
	IDTimestampScale   = 0x2AD7B1
	IDDuration         = 0x4489
	IDDateUTC          = 0x4461
	IDTitle            = 0x7BA9
	IDMuxingApp        = 0x4D80
	IDWritingApp       = 0x5741

	// Track
	IDTracks     = 0x1654AE6B
	IDTrackEntry = 0xAE
	IDTrackNum   = 0xD7
	IDTrackUID   = 0x73C5
	IDTrackType  = 0x83
	IDTrackName  = 0x536E
	IDLanguage   = 0x22B59C
	IDCodecID    = 0x86
	IDCodecPriv  = 0x63A2
	IDCodecName  = 0x258688
	IDVideo      = 0xE0
	IDAudio      = 0xE1

	// Video
	IDFlagInterlaced = 0x9A
	IDPixelWidth     = 0xB0
	IDPixelHeight    = 0xBA
	IDDisplayWidth   = 0x54B0
	IDDisplayHeight  = 0x54BA

	// Audio
	IDSamplingFrequency       = 0xB5
	IDOutputSamplingFrequency = 0x78B5
	IDChannels                = 0x9F
	IDBitDepth                = 0x6264

	// Cluster
	IDCluster     = 0x1F43B675
	IDTimestamp   = 0xE7
	IDSimpleBlock = 0xA3
	IDBlockGroup  = 0xA0
	IDBlock       = 0xA1

	// Cues
	IDCues     = 0x1C53BB6B
	IDCuePoint = 0xBB
	IDCueTime  = 0xB3

	// Chapters
	IDChapters = 0x1043A770

	// Tags
	IDTags = 0x1254C367

	// Attachments
	IDAttachments = 0x1941A469
)

// EBMLElement represents an EBML element
type EBMLElement struct {
	ID   uint32
	Size uint64
	Data []byte
}

// EBMLReader provides methods for reading EBML data
type EBMLReader struct {
	r   io.ReadSeeker
	pos int64
}

// NewEBMLReader creates a new EBML reader
func NewEBMLReader(r io.ReadSeeker) *EBMLReader {
	return &EBMLReader{r: r}
}

// ReadVInt reads a variable-length integer
func (er *EBMLReader) ReadVInt() (uint64, error) {
	return er.readVInt(false)
}

// ReadVIntID reads a variable-length integer for element IDs (keeps length marker)
func (er *EBMLReader) ReadVIntID() (uint64, error) {
	return er.readVInt(true)
}

// readVInt reads a variable-length integer
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

// ReadElement reads an EBML element
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

// Seek moves the reader to the specified position
func (er *EBMLReader) Seek(offset int64, whence int) (int64, error) {
	pos, err := er.r.Seek(offset, whence)
	if err != nil {
		return 0, err
	}
	er.pos = pos
	return pos, nil
}

// Position returns the current position
func (er *EBMLReader) Position() int64 {
	return er.pos
}

// ReadUInt reads an unsigned integer from element data
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

// ReadFloat reads a floating-point number from element data
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

// ReadString reads a UTF-8 string from element data
func (el *EBMLElement) ReadString() string {
	// Remove null terminator if present
	data := el.Data
	if len(data) > 0 && data[len(data)-1] == 0 {
		data = data[:len(data)-1]
	}
	return string(data)
}

// ReadBytes returns the raw element data
func (el *EBMLElement) ReadBytes() []byte {
	return el.Data
}

// SkipElement skips the current element by seeking past its data
func (er *EBMLReader) SkipElement(element *EBMLElement) error {
	_, err := er.r.Seek(int64(element.Size), io.SeekCurrent)
	if err != nil {
		return err
	}
	er.pos += int64(element.Size)
	return nil
}

// ReadElementHeader reads only the element ID and size, not the data
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

// EBMLHeader represents the EBML header
type EBMLHeader struct {
	Version            uint64
	ReadVersion        uint64
	MaxIDLength        uint64
	MaxSizeLength      uint64
	DocType            string
	DocTypeVersion     uint64
	DocTypeReadVersion uint64
}

// ReadEBMLHeader reads and parses the EBML header
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

// seekableReader wraps a bytes.Reader to implement io.ReadSeeker
type seekableReader struct {
	*bytes.Reader
}

func (sr *seekableReader) Seek(offset int64, whence int) (int64, error) {
	return sr.Reader.Seek(offset, whence)
}
