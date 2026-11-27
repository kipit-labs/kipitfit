// Package media keeps exercise pictures in an S3-compatible bucket (RustFS,
// MinIO, R2, AWS…). The bucket stays private: the app streams from it.
package media

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

var ErrNotFound = errors.New("media not found")

// Info describes a stored object.
type Info struct {
	Size        int64
	ContentType string
	ETag        string
	ModTime     time.Time
}

// Object is an open object: seekable, so the API can answer range requests.
type Object interface {
	io.ReadSeekCloser
}

type Store interface {
	// Stat returns ErrNotFound when key isn't there.
	Stat(ctx context.Context, key string) (Info, error)
	Get(ctx context.Context, key string) (Object, Info, error)
	Put(ctx context.Context, key string, r io.Reader, size int64, contentType string) error
}

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
func NewS3(c Config) (Store, error) {
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

func (s *s3Store) Stat(ctx context.Context, key string) (Info, error) {
	st, err := s.cl.StatObject(ctx, s.bucket, key, minio.StatObjectOptions{})
	if err != nil {
		return Info{}, mapErr(err)
	}
	return info(st), nil
}

func (s *s3Store) Get(ctx context.Context, key string) (Object, Info, error) {
	obj, err := s.cl.GetObject(ctx, s.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, Info{}, mapErr(err)
	}
	// GetObject is lazy: Stat makes the request and reports a missing key.
	st, err := obj.Stat()
	if err != nil {
		_ = obj.Close()
		return nil, Info{}, mapErr(err)
	}
	return obj, info(st), nil
}

func (s *s3Store) Put(ctx context.Context, key string, r io.Reader, size int64, contentType string) error {
	_, err := s.cl.PutObject(ctx, s.bucket, key, r, size, minio.PutObjectOptions{ContentType: contentType})
	return err
}

func info(st minio.ObjectInfo) Info {
	return Info{Size: st.Size, ContentType: st.ContentType, ETag: st.ETag, ModTime: st.LastModified}
}

func mapErr(err error) error {
	switch minio.ToErrorResponse(err).Code {
	case minio.NoSuchKey, "NotFound":
		return ErrNotFound
	}
	return err
}

// ValidKey reports whether key may be served: pack media only (never other
// things the bucket may hold later), no empty or dot segments.
func ValidKey(key string) bool {
	if !strings.HasPrefix(key, "packs/") || len(key) > 512 {
		return false
	}
	for _, seg := range strings.Split(key, "/") {
		if seg == "" || seg == "." || seg == ".." {
			return false
		}
		for _, r := range seg {
			if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') && r != '-' && r != '_' && r != '.' {
				return false
			}
		}
	}
	return true
}

// Memory is an in-memory Store for tests.
type Memory struct {
	mu   sync.Mutex
	objs map[string]memObj
	Puts int
}

type memObj struct {
	data []byte
	info Info
}

func NewMemory() *Memory { return &Memory{objs: map[string]memObj{}} }

func (m *Memory) Stat(_ context.Context, key string) (Info, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	o, ok := m.objs[key]
	if !ok {
		return Info{}, ErrNotFound
	}
	return o.info, nil
}

func (m *Memory) Get(_ context.Context, key string) (Object, Info, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	o, ok := m.objs[key]
	if !ok {
		return nil, Info{}, ErrNotFound
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
	m.objs[key] = memObj{b, Info{Size: int64(len(b)), ContentType: contentType, ETag: key, ModTime: time.Unix(0, 0)}}
	return nil
}

type nopCloser struct{ *bytes.Reader }

func (nopCloser) Close() error { return nil }
