package matroska

import (
	"fmt"
	"io"
)

// Demuxer is a Matroska demuxer using pure Go implementation.
type Demuxer struct {
	parser *MatroskaParser
	reader io.ReadSeeker
}

// NewDemuxer creates a new Matroska demuxer from r.
func NewDemuxer(r io.ReadSeeker) (*Demuxer, error) {
	parser, err := NewMatroskaParser(r, false)
	if err != nil {
		return nil, fmt.Errorf("failed to create parser: %w", err)
	}

	return &Demuxer{
		parser: parser,
		reader: r,
	}, nil
}

// NewStreamingDemuxer creates a new Matroska demuxer from an
// io.Reader that has no ability to seek on the input stream.
func NewStreamingDemuxer(r io.Reader) (*Demuxer, error) {
	fs := &fakeSeeker{r: r}
	parser, err := NewMatroskaParser(fs, true)
	if err != nil {
		return nil, fmt.Errorf("failed to create streaming parser: %w", err)
	}

	return &Demuxer{
		parser: parser,
		reader: fs,
	}, nil
}

// Close closes a demuxer.
func (d *Demuxer) Close() {
	// Pure Go implementation doesn't need explicit cleanup
}

// GetNumTracks gets the number of tracks available to a given demuxer.
func (d *Demuxer) GetNumTracks() (uint, error) {
	return d.parser.GetNumTracks(), nil
}

// GetTrackInfo returns all track-level information available for a given track,
// where track is less than what is returned by GetNumTracks.
func (d *Demuxer) GetTrackInfo(track uint) (*TrackInfo, error) {
	trackInfo := d.parser.GetTrackInfo(track)
	if trackInfo == nil {
		return nil, fmt.Errorf("track %d not found", track)
	}
	return trackInfo, nil
}

// GetFileInfo gets all top-level (whole file) info available for a given
// demuxer.
func (d *Demuxer) GetFileInfo() (*SegmentInfo, error) {
	fileInfo := d.parser.GetFileInfo()
	if fileInfo == nil {
		return nil, fmt.Errorf("no file info available")
	}
	return fileInfo, nil
}

// GetAttachments returns information on all available attachments
// for a given demuxer. The returned slice may be of length 0.
func (d *Demuxer) GetAttachments() []*Attachment {
	return d.parser.GetAttachments()
}

// GetChapters returns all chapters for a given demuxer. The returned slice may
// be of length 0.
func (d *Demuxer) GetChapters() []*Chapter {
	return d.parser.GetChapters()
}

// GetTags returns all tags for a given demuxer. The returned slice may be of
// length 0.
func (d *Demuxer) GetTags() []*Tag {
	return d.parser.GetTags()
}

// GetCues returns all cues for a given demuxer. The returned slice may be
// of length 0.
func (d *Demuxer) GetCues() []*Cue {
	return d.parser.GetCues()
}

// GetSegment returns the position of the segment.
func (d *Demuxer) GetSegment() uint64 {
	return d.parser.GetSegment()
}

// GetSegmentTop returns the position of the next byte after the segment.
func (d *Demuxer) GetSegmentTop() uint64 {
	return d.parser.GetSegmentTop()
}

// GetCuesPos returna the position of the cues in the stream.
func (d *Demuxer) GetCuesPos() uint64 {
	return d.parser.GetCuesPos()
}

// GetCuesTopPos returns the position of the byte after the end of the cues.
func (d *Demuxer) GetCuesTopPos() uint64 {
	return d.parser.GetCuesTopPos()
}

// Seek seeks to a given timecode.
//
// Flags here may be: 0 (normal seek), matroska.SeekToPrevKeyFrame,
// or matoska.SeekToPrevKeyFrameStrict
func (d *Demuxer) Seek(timecode uint64, flags uint32) {
	// TODO: Implement seeking in pure Go parser
}

// SeekCueAware seeks to a given timecode while taking cues into account
//
// Flags here may be: 0 (normal seek), matroska.SeekToPrevKeyFrame,
// or matoska.SeekToPrevKeyFrameStrict
//
// fuzzy defines whether a fuzzy seek will be used or not.
func (d *Demuxer) SeekCueAware(timecode uint64, flags uint32, fuzzy bool) {
	// TODO: Implement cue-aware seeking in pure Go parser
}

// SkipToKeyframe skips to the next keyframe in a stream.
func (d *Demuxer) SkipToKeyframe() {
	// TODO: Implement keyframe skipping in pure Go parser
}

// GetLowestQTimecode returns the lowest queued timecode in the demuxer.
func (d *Demuxer) GetLowestQTimecode() uint64 {
	// TODO: Implement timecode tracking in pure Go parser
	return 0
}

// SetTrackMask sets the demuxer's track mask; that is, it tells the demuxer
// which tracks to skip, and which to use. Any tracks with ones in their bit
// positions will be ignored.
//
// Calling this withh cause all parsed and queued frames to be discarded.
func (d *Demuxer) SetTrackMask(mask uint64) {
	// TODO: Implement track masking in pure Go parser
}

// ReadPacketMask is the same as ReadPacket except with a track mask.
func (d *Demuxer) ReadPacketMask(mask uint64) (*Packet, error) {
	// For now, ignore mask and read next packet
	return d.parser.ReadPacket()
}

// ReadPacket returns the next packet from a demuxer.
func (d *Demuxer) ReadPacket() (*Packet, error) {
	return d.parser.ReadPacket()
}
