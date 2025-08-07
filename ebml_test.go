package matroska

import (
	"bytes"
	"encoding/binary"
	"io"
	"math"
	"reflect"
	"testing"
)

// TestReadVInt tests the readVInt function with various inputs.
func TestReadVInt(t *testing.T) {
	testCases := []struct {
		name             string
		input            []byte
		keepLengthMarker bool
		expectedVal      uint64
		expectErr        bool
	}{
		// 1-byte VINTs
		{"1-byte value", []byte{0x81}, false, 1, false},
		{"1-byte max value", []byte{0xFF}, false, 127, false},
		{"1-byte with length marker", []byte{0x81}, true, 0x81, false},

		// 2-byte VINTs
		{"2-byte value", []byte{0x40, 0x01}, false, 1, false},
		{"2-byte value high", []byte{0x50, 0x11}, false, 0x1011, false},
		{"2-byte max value", []byte{0x7F, 0xFF}, false, (1 << 14) - 1, false},
		{"2-byte with length marker", []byte{0x50, 0x11}, true, 0x5011, false},

		// 4-byte VINTs
		{"4-byte value", []byte{0x10, 0x00, 0x00, 0x01}, false, 1, false},
		{"4-byte value high", []byte{0x1A, 0xBC, 0xDE, 0xF0}, false, 0xABCDEF0, false},
		{"4-byte max value", []byte{0x1F, 0xFF, 0xFF, 0xFF}, false, (1 << 28) - 1, false},
		{"4-byte with length marker", []byte{0x1A, 0xBC, 0xDE, 0xF0}, true, 0x1ABCDEF0, false},

		// 8-byte VINTs
		{"8-byte value", []byte{0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01}, false, 1, false},
		{"8-byte value high", []byte{0x01, 0x23, 0x45, 0x67, 0x89, 0xAB, 0xCD, 0xEF}, false, 0x23456789ABCDEF, false},
		{"8-byte max value", []byte{0x01, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}, false, (1 << 56) - 1, false},
		{"8-byte with length marker", []byte{0x01, 0x23, 0x45, 0x67, 0x89, 0xAB, 0xCD, 0xEF}, true, 0x0123456789ABCDEF, false},

		// Error cases
		{"invalid VINT zero byte", []byte{0x00}, false, 0, true},
		{"EOF in second byte", []byte{0x40}, false, 0, true},
		{"EOF in later byte", []byte{0x10, 0x00}, false, 0, true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			r := bytes.NewReader(tc.input)
			reader := NewEBMLReader(r)

			val, err := reader.readVInt(tc.keepLengthMarker)

			if tc.expectErr {
				if err == nil {
					t.Errorf("Expected an error, but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if val != tc.expectedVal {
					t.Errorf("Expected value %d, but got %d", tc.expectedVal, val)
				}
			}
		})
	}
}

// TestEBMLElementRead_Types tests the type reading methods of EBMLElement.
func TestEBMLElementRead_Types(t *testing.T) {
	t.Run("ReadUInt", func(t *testing.T) {
		el := &EBMLElement{Data: []byte{0x01, 0x02, 0x03, 0x04}}
		expected := uint64(0x01020304)
		if val := el.ReadUInt(); val != expected {
			t.Errorf("ReadUInt() = %v, want %v", val, expected)
		}
	})

	t.Run("ReadInt", func(t *testing.T) {
		// Positive
		elPos := &EBMLElement{Data: []byte{0x01, 0x02, 0x03, 0x04}}
		expectedPos := int64(0x01020304)
		if val := elPos.ReadInt(); val != expectedPos {
			t.Errorf("ReadInt() positive = %v, want %v", val, expectedPos)
		}
		// Negative
		elNeg := &EBMLElement{Data: []byte{0xFF, 0xFF, 0xFF, 0xFE}} // -2 in 4 bytes
		expectedNeg := int64(-2)
		if val := elNeg.ReadInt(); val != expectedNeg {
			t.Errorf("ReadInt() negative = %v, want %v", val, expectedNeg)
		}
	})

	t.Run("ReadFloat", func(t *testing.T) {
		// 32-bit float
		var f32 float32 = 3.14
		bits32 := math.Float32bits(f32)
		data32 := make([]byte, 4)
		binary.BigEndian.PutUint32(data32, bits32)
		el32 := &EBMLElement{Data: data32}
		if val := el32.ReadFloat(); float32(val) != f32 {
			t.Errorf("ReadFloat() 32-bit = %v, want %v", val, f32)
		}

		// 64-bit float
		var f64 float64 = 3.1415926535
		bits64 := math.Float64bits(f64)
		data64 := make([]byte, 8)
		binary.BigEndian.PutUint64(data64, bits64)
		el64 := &EBMLElement{Data: data64}
		if val := el64.ReadFloat(); val != f64 {
			t.Errorf("ReadFloat() 64-bit = %v, want %v", val, f64)
		}
	})

	t.Run("ReadString", func(t *testing.T) {
		el := &EBMLElement{Data: []byte("hello")}
		if val := el.ReadString(); val != "hello" {
			t.Errorf("ReadString() = %q, want %q", val, "hello")
		}
		// With null terminator
		elNull := &EBMLElement{Data: []byte("hello\x00")}
		if val := elNull.ReadString(); val != "hello" {
			t.Errorf("ReadString() with null = %q, want %q", val, "hello")
		}
	})

	t.Run("ReadBytes", func(t *testing.T) {
		data := []byte{1, 2, 3}
		el := &EBMLElement{Data: data}
		if val := el.ReadBytes(); !reflect.DeepEqual(val, data) {
			t.Errorf("ReadBytes() = %v, want %v", val, data)
		}
	})
}

// TestEBMLReader_ReadElement tests reading a full element.
func TestEBMLReader_ReadElement(t *testing.T) {
	// ID: 0x1A45DFA3 (EBMLHeader), Size: 4, Data: "test"
	input := []byte{0x1A, 0x45, 0xDF, 0xA3, 0x84, 't', 'e', 's', 't'}
	r := bytes.NewReader(input)
	reader := NewEBMLReader(r)

	el, err := reader.ReadElement()
	if err != nil {
		t.Fatalf("ReadElement() failed: %v", err)
	}

	if el.ID != IDEBMLHeader {
		t.Errorf("Expected ID 0x%X, got 0x%X", IDEBMLHeader, el.ID)
	}
	if el.Size != 4 {
		t.Errorf("Expected size 4, got %d", el.Size)
	}
	if string(el.Data) != "test" {
		t.Errorf("Expected data 'test', got %q", string(el.Data))
	}
}

// TestEBMLReader_ReadEBMLHeader tests parsing the EBML header.
func TestEBMLReader_ReadEBMLHeader(t *testing.T) {
	// EBMLHeader (ID 0x1A45DFA3)
	//   - EBMLVersion (ID 0x4286), Size 1, Value 1
	//   - DocType (ID 0x4282), Size 8, Value "matroska"
	headerData := []byte{
		0x42, 0x86, 0x81, 0x01, // EBMLVersion
		0x42, 0x82, 0x88, 'm', 'a', 't', 'r', 'o', 's', 'k', 'a', // DocType
	}
	headerSize := len(headerData)

	buf := new(bytes.Buffer)
	// Write EBML Header ID
	buf.Write([]byte{0x1A, 0x45, 0xDF, 0xA3})
	// Write size
	buf.Write([]byte{byte(0x80 | headerSize)})
	// Write data
	buf.Write(headerData)

	r := bytes.NewReader(buf.Bytes())
	reader := NewEBMLReader(r)

	header, err := reader.ReadEBMLHeader()
	if err != nil {
		t.Fatalf("ReadEBMLHeader() failed: %v", err)
	}
	if header.Version != 1 {
		t.Errorf("Expected Version 1, got %d", header.Version)
	}
	if header.DocType != "matroska" {
		t.Errorf("Expected DocType 'matroska', got %q", header.DocType)
	}
}

func TestEBMLReader_ReadElementHeader(t *testing.T) {
	input := []byte{0x1A, 0x45, 0xDF, 0xA3, 0x84, 't', 'e', 's', 't'}
	r := bytes.NewReader(input)
	reader := NewEBMLReader(r)

	id, size, err := reader.ReadElementHeader()
	if err != nil {
		t.Fatalf("ReadElementHeader() failed: %v", err)
	}

	if id != IDEBMLHeader {
		t.Errorf("Expected ID 0x%X, got 0x%X", IDEBMLHeader, id)
	}
	if size != 4 {
		t.Errorf("Expected size 4, got %d", size)
	}

	// Check that we can read the rest of the data
	data := make([]byte, size)
	n, err := io.ReadFull(reader.r, data)
	if err != nil {
		t.Fatalf("Failed to read data after header: %v", err)
	}
	if n != 4 {
		t.Errorf("Expected to read 4 bytes, got %d", n)
	}
	if string(data) != "test" {
		t.Errorf("Expected data 'test', got %q", string(data))
	}
}

func TestEBMLReader_SkipElement(t *testing.T) {
	input := []byte{
		// First element: ID: 0x4286, Size: 1, Data: 1
		0x42, 0x86, 0x81, 0x01,
		// Second element: ID: 0x4282, Size: 8, Data: "matroska"
		0x42, 0x82, 0x88, 'm', 'a', 't', 'r', 'o', 's', 'k', 'a',
	}
	r := bytes.NewReader(input)
	reader := NewEBMLReader(r)

	// Read header of the first element
	id1, size1, err := reader.ReadElementHeader()
	if err != nil {
		t.Fatalf("Failed to read first element header: %v", err)
	}

	// Skip the data of the first element
	el1ToSkip := &EBMLElement{ID: uint32(id1), Size: size1}
	err = reader.SkipElement(el1ToSkip)
	if err != nil {
		t.Fatalf("SkipElement() failed: %v", err)
	}

	// Now, the reader should be at the start of the second element.
	// Let's read the second element to verify.
	el2, err := reader.ReadElement()
	if err != nil {
		t.Fatalf("Failed to read second element after skip: %v", err)
	}
	if el2.ID != IDEBMLDocType { // 0x4282
		t.Errorf("Expected second element ID 0x%X, got 0x%X", IDEBMLDocType, el2.ID)
	}
	if el2.ReadString() != "matroska" {
		t.Errorf("Expected second element data 'matroska', got %q", el2.ReadString())
	}
}
