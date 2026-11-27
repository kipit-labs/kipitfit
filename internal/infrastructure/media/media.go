// Package media keeps exercise pictures in an S3-compatible bucket (RustFS,
// MinIO, R2, AWS…). The bucket stays private: the app streams from it.
package media

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/url"
	"sync"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"github.com/kipit-labs/kipitfit/internal/domain"
)

// Config is where the bucket is.
type Config struct {
	// Endpoint is the S3 URL, e.g. http://localhost:9000 or https://s3.example.com.
	Endpoint  string
	Bucket    string
	AccessKey string
	SecretKey string
	Region    string
}

// NewS3 connects to the bucket, with path-style URLs so RustFS and MinIO work
// without wildcard DNS.
func NewS3(c Config) (domain.MediaStore, error) {
	u, err := url.Parse(c.Endpoint)
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
		return nil, errors.New("S3 endpoint must be an http(s) URL")
	}
	cl, err := minio.New(u.Host, &minio.Options{
		Creds:        credentials.NewStaticV4(c.AccessKey, c.SecretKey, ""),
		Secure:       u.Scheme == "https",
		Region:       c.Region,
		BucketLookup: minio.BucketLookupPath,
	})
	if err != nil {
		return nil, err
	}
	return &s3Store{cl: cl, bucket: c.Bucket}, nil
}

type s3Store struct {
	cl     *minio.Client
	bucket string
}

func (s *s3Store) Stat(ctx context.Context, key string) (domain.MediaInfo, error) {
	st, err := s.cl.StatObject(ctx, s.bucket, key, minio.StatObjectOptions{})
	if err != nil {
		return domain.MediaInfo{}, mapErr(err)
	}
	return info(st), nil
}

func (s *s3Store) Get(ctx context.Context, key string) (io.ReadSeekCloser, domain.MediaInfo, error) {
	obj, err := s.cl.GetObject(ctx, s.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, domain.MediaInfo{}, mapErr(err)
	}
	// GetObject is lazy: Stat makes the request and reports a missing key.
	st, err := obj.Stat()
	if err != nil {
		_ = obj.Close()
		return nil, domain.MediaInfo{}, mapErr(err)
	}
	return obj, info(st), nil
}

func (s *s3Store) Put(ctx context.Context, key string, r io.Reader, size int64, contentType string) error {
	_, err := s.cl.PutObject(ctx, s.bucket, key, r, size, minio.PutObjectOptions{ContentType: contentType})
	return err
}

func info(st minio.ObjectInfo) domain.MediaInfo {
	return domain.MediaInfo{Size: st.Size, ContentType: st.ContentType, ETag: st.ETag, ModTime: st.LastModified}
}

func mapErr(err error) error {
	switch minio.ToErrorResponse(err).Code {
	case minio.NoSuchKey, "NotFound":
		return domain.ErrNotFound
	}
	return err
}

// Memory is an in-memory domain.MediaStore for tests.
type Memory struct {
	mu   sync.Mutex
	objs map[string]memObj
	Puts int
}

type memObj struct {
	data []byte
	info domain.MediaInfo
}

func NewMemory() *Memory { return &Memory{objs: map[string]memObj{}} }

func (m *Memory) Stat(_ context.Context, key string) (domain.MediaInfo, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	o, ok := m.objs[key]
	if !ok {
		return domain.MediaInfo{}, domain.ErrNotFound
	}
	return o.info, nil
}

func (m *Memory) Get(_ context.Context, key string) (io.ReadSeekCloser, domain.MediaInfo, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	o, ok := m.objs[key]
	if !ok {
		return nil, domain.MediaInfo{}, domain.ErrNotFound
	}
	return nopCloser{bytes.NewReader(o.data)}, o.info, nil
}

func (m *Memory) Put(_ context.Context, key string, r io.Reader, _ int64, contentType string) error {
	b, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Puts++
	m.objs[key] = memObj{b, domain.MediaInfo{Size: int64(len(b)), ContentType: contentType, ETag: key, ModTime: time.Unix(0, 0)}}
	return nil
}

type nopCloser struct{ *bytes.Reader }

func (nopCloser) Close() error { return nil }
