package runninghub

import (
	"context"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
)

// UploadBinaryFile calls POST /openapi/v2/media/upload/binary.
// It streams the file as multipart/form-data with form name `file`.
func (c *Client) UploadBinaryFile(ctx context.Context, filePath string) (*UploadBinaryData, error) {
	pr, pw := io.Pipe()
	mw := multipart.NewWriter(pw)

	go func() {
		defer func() {
			_ = mw.Close()
			_ = pw.Close()
		}()

		f, err := os.Open(filePath)
		if err != nil {
			_ = pw.CloseWithError(err)
			return
		}
		defer f.Close()

		part, err := mw.CreateFormFile("file", filepath.Base(filePath))
		if err != nil {
			_ = pw.CloseWithError(err)
			return
		}
		if _, err := io.Copy(part, f); err != nil {
			_ = pw.CloseWithError(err)
			return
		}
	}()

	hdr := make(http.Header)
	hdr.Set("Content-Type", mw.FormDataContentType())

	var resp MessageEnvelope[UploadBinaryData]
	if err := c.doRaw(ctx, http.MethodPost, "/openapi/v2/media/upload/binary", nil, hdr, pr, &resp); err != nil {
		return nil, err
	}
	if resp.Code != 0 {
		return nil, &APIError{Code: resp.Code, Message: resp.Message}
	}
	if resp.Data == nil {
		return nil, &APIError{Code: resp.Code, Message: "empty data"}
	}
	return resp.Data, nil
}

// UploadBinaryReader uploads from an arbitrary reader with a filename.
func (c *Client) UploadBinaryReader(ctx context.Context, r io.Reader, filename string) (*UploadBinaryData, error) {
	pr, pw := io.Pipe()
	mw := multipart.NewWriter(pw)

	go func() {
		defer func() {
			_ = mw.Close()
			_ = pw.Close()
		}()

		part, err := mw.CreateFormFile("file", filepath.Base(filename))
		if err != nil {
			_ = pw.CloseWithError(err)
			return
		}
		if _, err := io.Copy(part, r); err != nil {
			_ = pw.CloseWithError(err)
			return
		}
	}()

	hdr := make(http.Header)
	hdr.Set("Content-Type", mw.FormDataContentType())

	var resp MessageEnvelope[UploadBinaryData]
	if err := c.doRaw(ctx, http.MethodPost, "/openapi/v2/media/upload/binary", nil, hdr, pr, &resp); err != nil {
		return nil, err
	}
	if resp.Code != 0 {
		return nil, &APIError{Code: resp.Code, Message: resp.Message}
	}
	if resp.Data == nil {
		return nil, &APIError{Code: resp.Code, Message: "empty data"}
	}
	return resp.Data, nil
}
