// Package tiff implements TIFF decoding as defined in TIFF 6.0 specification at
// http://partners.adobe.com/public/developer/en/tiff/TIFF6.pdf
package tiff

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
)

// ReadAtReader is used when decoding Tiff tags and directories
type ReadAtReader interface {
	io.Reader
	io.ReaderAt
}

// Tiff provides access to a decoded tiff data structure.
type Tiff struct {
	// Dirs is an ordered slice of the tiff's Image File Directories (IFDs).
	// The IFD at index 0 is IFD0.
	Dirs []*Dir
	// The tiff's byte-encoding (i.e. big/little endian).
	Order binary.ByteOrder
}

func DecodeFile(f *os.File) (*Tiff, error) {
	t := new(Tiff)

	// read byte order
	bo := make([]byte, 2)
  if _, err := io.ReadFull(f, bo); err != nil {
		return nil, errors.New("tiff: could not read tiff byte order")
	}
	if string(bo) == "II" {
		t.Order = binary.LittleEndian
	} else if string(bo) == "MM" {
		t.Order = binary.BigEndian
	} else {
		return nil, errors.New("tiff: could not read tiff byte order")
	}

	// check for special tiff marker
  spbuf := make([]byte, 2)
  if _, err := io.ReadFull(f, spbuf); err != nil {
		return nil, errors.New("tiff: could not read special tiff marker")
  }
	var sp int16
  err := binary.Read(bytes.NewReader(spbuf), t.Order, &sp)
	if err != nil || 42 != sp {
		return nil, errors.New("tiff: could not find special tiff marker")
	}

	// load offset to first IFD
  offsetbuf := make([]byte, 4)
  if _, err = io.ReadFull(f, offsetbuf); err != nil {
    return nil, errors.New("tiff: could not read offset of 1st IFD")
  }
	var offset0 uint32
	err = binary.Read(bytes.NewReader(offsetbuf), t.Order, &offset0)
	if err != nil {
		return nil, errors.New("tiff: could not read offset to first IFD")
	}

  // seek to offset
  _, err = f.Seek(int64(offset0), 0)
  if err != nil {
    return nil, errors.New("tiff: seek to IFD failed")
  }

  // fmt.Printf("Seeking to %d\n", offset0)

  // metadata is at the end from experimentation so it is fine to read the rest of the file now
  data, err := ioutil.ReadAll(f)
	buf := bytes.NewReader(data)

	// load IFD's
	var d *Dir
	prev := offset0
  next_offset := offset0
	for next_offset != 0 {
		// load the dir
		d, next_offset, err = DecodeDir(buf, t.Order, offset0)

		if err != nil {
			return nil, err
		}

		if next_offset == prev {
			return nil, errors.New("tiff: recursive IFD")
		}
		prev = next_offset

		t.Dirs = append(t.Dirs, d)
	}

	return t, nil
}

// Decode parses tiff-encoded data from r and returns a Tiff struct that
// reflects the structure and content of the tiff data. The first read from r
// should be the first byte of the tiff-encoded data and not necessarily the
// first byte of an os.File object.
func Decode(r io.Reader) (*Tiff, error) {
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, errors.New("tiff: could not read data")
	}
	buf := bytes.NewReader(data)

	t := new(Tiff)

	// read byte order
	bo := make([]byte, 2)
	if _, err = io.ReadFull(buf, bo); err != nil {
		return nil, errors.New("tiff: could not read tiff byte order")
	}
	if string(bo) == "II" {
		t.Order = binary.LittleEndian
	} else if string(bo) == "MM" {
		t.Order = binary.BigEndian
	} else {
		return nil, errors.New("tiff: could not read tiff byte order")
	}

	// check for special tiff marker
	var sp int16
	err = binary.Read(buf, t.Order, &sp)
	if err != nil || 42 != sp {
		return nil, errors.New("tiff: could not find special tiff marker")
	}

	// load offset to first IFD
	var offset uint32
	err = binary.Read(buf, t.Order, &offset)
	if err != nil {
		return nil, errors.New("tiff: could not read offset to first IFD")
	}

	// load IFD's
	var d *Dir
	prev := offset
	for offset != 0 {
		// seek to offset
		_, err := buf.Seek(int64(offset), 0)
		if err != nil {
			return nil, errors.New("tiff: seek to IFD failed")
		}

		if buf.Len() == 0 {
			return nil, errors.New("tiff: seek offset after EOF")
		}

		// load the dir
		d, offset, err = DecodeDir(buf, t.Order, 0)
		if err != nil {
			return nil, err
		}

		if offset == prev {
			return nil, errors.New("tiff: recursive IFD")
		}
		prev = offset

		t.Dirs = append(t.Dirs, d)
	}

	return t, nil
}

func (tf *Tiff) String() string {
	var buf bytes.Buffer
	fmt.Fprint(&buf, "Tiff{")
	for _, d := range tf.Dirs {
		fmt.Fprintf(&buf, "%s, ", d.String())
	}
	fmt.Fprintf(&buf, "}")
	return buf.String()
}

// Dir provides access to the parsed content of a tiff Image File Directory (IFD).
type Dir struct {
	Tags []*Tag
}

// DecodeDir parses a tiff-encoded IFD from r and returns a Dir object.  offset
// is the offset to the next IFD.  The first read from r should be at the first
// byte of the IFD. ReadAt offsets should generally be relative to the
// beginning of the tiff structure (not relative to the beginning of the IFD).
// position_offset is where the first byte of r starts relative to the start of the file
func DecodeDir(r ReadAtReader, order binary.ByteOrder, position_offset uint32) (d *Dir, offset uint32, err error) {
	d = new(Dir)

	// get num of tags in ifd
	var nTags int16
	err = binary.Read(r, order, &nTags)
	if err != nil {
		return nil, 0, errors.New("tiff: failed to read IFD tag count: " + err.Error())
	}

	// load tags
	for n := 0; n < int(nTags); n++ {
		t, err := DecodeTag(r, order, position_offset)
		if err != nil {
			return nil, 0, err
		}
		d.Tags = append(d.Tags, t)
	}

	// get offset to next ifd
	err = binary.Read(r, order, &offset)
	if err != nil {
		return nil, 0, errors.New("tiff: falied to read offset to next IFD: " + err.Error())
	}

	return d, uint32(offset), nil
}

func (d *Dir) String() string {
	s := "Dir{"
	for _, t := range d.Tags {
		s += t.String() + ", "
	}
	return s + "}"
}
