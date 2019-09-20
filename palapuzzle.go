// palapuzzle is a package for reading .puzzle files created by/for Palapeli

package palapuzzle

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// A PuzzleInfo holds the interesting details from a .puzzle file
type PuzzleInfo struct {
	Filename, Dir string
	Title         string
	Author        string
	Comment       string
	Warnings      []string
	NPieceFiles   int
	NPiecesDecl   int
	ImageFileSize int64
}

var rePieceName = regexp.MustCompile(`^(\d+)\.png$`)
var reKeyValue = regexp.MustCompile(`^([^[=]+)=(.*)$`)

// ScanPuzzle() reads a .puzzle file, does some checking and returns a
// PuzzleInfo or an error (but not both).
func ScanPuzzle(fs string) (*PuzzleInfo, error) {
	var ret = &PuzzleInfo{}

	f, err := os.Open(fs)
	if err != nil {
		return nil, &Error{"cannot open", fs, err}
	}
	defer f.Close()
	ret.Dir, ret.Filename = filepath.Split(fs)

	zr, err := gzip.NewReader(f)
	if err != nil {
		return nil, &Error{"cannot decompress", fs, err}
	}
	defer zr.Close()

	tr := tar.NewReader(zr)
	var maxPieceNum = -1
	var piecesFound = make([]byte, 512)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, &Error{"cannot read decompressed TAR file", fs, err}
		}
		if m := rePieceName.FindStringSubmatch(header.Name); m != nil {
			i, err := strconv.Atoi(m[1])
			if err != nil {
				text := fmt.Sprintf("bad member name %q", header.Name)
				return nil, &Error{text, fs, err}
			}
			length := len(piecesFound)
			if i >= length {
				newSlice := make([]byte, 2*i)
				copy(newSlice, piecesFound)
				piecesFound = newSlice
			}
			piecesFound[i]++
			if i > maxPieceNum {
				maxPieceNum = i
			}
		} else if header.Name == "image.jpg" {
			ret.ImageFileSize = header.Size
		} else if header.Name == "pala.desktop" {
			if e := scanPalaDesktopFile(tr, ret); e != nil {
				e.FilePath = fs
				return nil, e
			}
		}
	}
	for i := 0; i < maxPieceNum; i++ {
		if piecesFound[i] == 0 {
			ret.Warnings = append(ret.Warnings,
				fmt.Sprintf(`missing "%d.png"`, i))
		} else if piecesFound[i] > 1 {
			ret.Warnings = append(ret.Warnings,
				fmt.Sprintf(`%d members named "%d.png"`,
					piecesFound[i], i))
		}
	}
	ret.NPieceFiles = maxPieceNum + 1

	return ret, nil
}

func scanPalaDesktopFile(tr io.Reader, out *PuzzleInfo) *Error {
	s := bufio.NewScanner(tr)
	for s.Scan() {
		if m := reKeyValue.FindStringSubmatch(s.Text()); m != nil {
			key, value := m[1], strings.TrimSpace(m[2])
			switch key {
			case "Name":
				out.Title = value
			case "X-KDE-PluginInfo-Author":
				out.Author = value
			case "Comment":
				out.Comment = value
			case "PieceCount", "020_PieceCount":
				n, err := strconv.Atoi(value)
				if err != nil {
					n = -1
					out.Warnings = append(out.Warnings,
						fmt.Sprintf("bad PieceCount %q", value))
				}
				out.NPiecesDecl = n
			}
		}
	}
	if s.Err() != nil {
		return &Error{`cannot read "pala.desktop" member in `, "?", s.Err()}
	}
	return nil
}

type Error struct { // Order must match ‘return &Error{"what", which, e}’ above
	Action    string // What we were trying to do
	FilePath  string // Which file we were trying to parse
	BaseError error  // Error from another package
}

func (e *Error) Error() string {
	be, baseErrStr := e.BaseError, ""
	if be2, ok := be.(*os.PathError); ok {
		be = be2.Err
	}
	baseErrStr = `: ` + be.Error()
	return `cannot ` + e.Action + ` "` + e.FilePath + `"` + baseErrStr
}
