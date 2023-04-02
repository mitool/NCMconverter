// Package ncm reads ncm file into memory, and cut it into different parts.
package ncm

import (
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
)

const (
	// MagicHeader1 is used to check whether the file is ncm format file.
	MagicHeader1 = 0x4e455443
	MagicHeader2 = 0x4d414446
)

type Data struct {
	Length uint64
	Detail []byte
}

// NcmFile represents a ncm file on the computer
type NcmFile struct {
	Path     string
	FileDir  string
	FileName string
	fd       *os.File
	Ext      string
	valid    bool
	Key      Data
	Meta     Data
	Cover    Data
	Music    Data
}

// NewNcmFile create a pointer to ncmfile
// ncmpath point out where the ncm file is.
func NewNcmFile(ncmpath string) (nf *NcmFile, err error) {
	nf = new(NcmFile)
	ncmpath = filepath.Clean(ncmpath)
	nf.FileName = filepath.Base(ncmpath)
	nf.FileDir = filepath.Dir(ncmpath)
	nf.Ext = filepath.Ext(ncmpath)
	nf.Path = ncmpath
	if nf.fd, err = os.Open(nf.Path); err != nil {
		return nil, err
	}
	return
}

// Validate check the whether the file is ncm file
// by ext and header.
func (nf *NcmFile) Validate() error {
	if !strings.EqualFold(nf.Ext, ".ncm") {
		return ErrExtNcm
	}

	err := nf.CheckHeader()

	if err != nil {
		return err
	}

	nf.valid = true
	return nil
}

// CheckHeader check whether file's first 4 bytes equal to MagicHeader1
// and the following 4 bytes equal to MagicHeader2.
func (nf *NcmFile) CheckHeader() error {
	if _, err := nf.fd.Seek(0, io.SeekStart); err != nil {
		return err
	}

	m1, err := readUint32(nf.fd)
	m2, err := readUint32(nf.fd)
	if err != nil {
		return err
	}
	if m1 != MagicHeader1 && m2 != MagicHeader2 {
		return ErrMagicHeader
	}
	return nil
}

// getData gets (a) uint32 format number as length,
// and the following (a) bytes data.
func (nf *NcmFile) getData(offset int64) ([]byte, uint32, error) {
	if _, err := nf.fd.Seek(offset, io.SeekStart); err != nil {
		return nil, 0, err
	}
	length, err := readUint32(nf.fd)
	if err != nil {
		return nil, 0, err
	}

	buf := make([]byte, length)

	if _, err = nf.fd.Read(buf); err != nil {
		return nil, 0, err
	}
	return buf, length, nil
}

// GetKey gets key used parse the music data.
func (nf *NcmFile) GetKey() (err error) {
	tmp, length, err := nf.getData(4*2 + 2)
	if err != nil {
		return err
	}
	nf.Key.Length = uint64(length)
	nf.Key.Detail = tmp
	return nil
}

// GetMeta gets meta data describing the music.
func (nf *NcmFile) GetMeta() (err error) {
	tmp, length, err := nf.getData(int64(4*2 + 2 + 4 + nf.Key.Length))
	if err != nil {
		return
	}
	nf.Meta.Detail = tmp
	nf.Meta.Length = uint64(length)
	return nil
}

// GetCover gets the cover of music.
func (nf *NcmFile) GetCover() (err error) {
	tmp, length, err := nf.getData(int64(4*2 + 2 + 4 + nf.Key.Length + 4 + nf.Meta.Length + 5 + 4))
	if err != nil {
		return
	}

	nf.Cover.Detail = tmp
	nf.Cover.Length = uint64(length)
	return nil
}

// GetMusicData gets the decoded music data.
func (nf *NcmFile) GetMusicData() error {
	nf.fd.Seek(int64(4*2+2+4+nf.Key.Length+4+nf.Meta.Length+9+4+nf.Cover.Length), io.SeekStart)
	file := nf.fd
	buf := make([]byte, 1024)
	nf.Music.Detail = make([]byte, 0)
	var (
		length int
		err    error
	)
	for {
		if length, err = file.Read(buf); err != nil && err != io.EOF {
			return err
		}
		nf.Music.Detail = append(nf.Music.Detail, buf[:length]...)
		nf.Music.Length += uint64(length)
		if err == io.EOF {
			return nil
		}
	}
}

// Close closes fd.
func (nf *NcmFile) Close() error {
	return nf.fd.Close()
}

// Parse is a combination of Validate,GetKey,GetMeta,GetCover and GetMusicData
// cause these methods have to be called in order except Validata.
func (nf *NcmFile) Parse() error {
	err := nf.Validate()
	if err != nil {
		log.Printf("Ncm magic header check failed: %v", err)
		return err
	}

	err = nf.GetKey()
	if err != nil {
		log.Printf("Get Key Failed: %v", err)
		return err
	}

	err = nf.GetMeta()
	if err != nil {
		log.Printf("Get Meta Failed: %v", err)
		return err
	}

	err = nf.GetCover()
	if err != nil {
		log.Printf("Get Cover Failed: %v", err)
		return err
	}

	err = nf.GetMusicData()
	if err != nil {
		log.Printf("Get Music Data Failed: %v", err)
		return err
	}

	return nil
}

func (nf *NcmFile) GetFDStat() (os.FileInfo, error) {
	return nf.fd.Stat()
}
