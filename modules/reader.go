package modules

import (
	"errors"
	"io"
	"mime/multipart"
)

type FormReader struct {
	*multipart.Reader
	part    *multipart.Part
	CanRead int64
}

func NewFormReader(reader io.Reader, boundary string, canRead int64) (*FormReader, error) {
	fp := &FormReader{
		Reader:  multipart.NewReader(reader, boundary),
		part:    nil,
		CanRead: canRead,
	}
	p, err := fp.NextPart()
	if err != nil {
		return nil, err
	}
	fp.part = p
	return fp, nil
}

func (r *FormReader) Read(p []byte) (n int, err error) {
	n, err = r.part.Read(p)
	r.CanRead -= int64(n)
	if r.CanRead < 0 {
		return n, errors.New("wrong file size")
	}
	return
}

func (r *FormReader) Close() error {
	for {
		_, err := r.NextPart()
		if err != nil {
			if err == io.EOF {
				return nil
			} else {
				return err
			}
		}

	}
}
