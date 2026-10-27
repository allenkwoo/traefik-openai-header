package traefik_openai_header

import (
	"bytes"
	"context"
	"io"
	"testing"
)

// The bufferedBody mutex is per request, so it is only ever contended by the
// transport's Read and Close on the same body. These measure its cost on the
// read path, serially and with requests running in parallel.

func benchHandler(b *testing.B) *Handler {
	h, err := New(context.Background(), &nextRecorder{}, CreateConfig(), "bench")
	if err != nil {
		b.Fatal(err)
	}
	return h.(*Handler)
}

func benchmarkBufferedBodyRead(b *testing.B, size int) {
	e := benchHandler(b)
	data := bytes.Repeat([]byte("a"), size)
	buf := make([]byte, 32<<10)
	b.SetBytes(int64(size))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		body := &bufferedBody{handler: e, data: data, reader: bytes.NewReader(data)}
		_, _ = io.CopyBuffer(io.Discard, body, buf)
		_ = body.Close()
	}
}

func BenchmarkBufferedBodyRead_1KiB(b *testing.B)  { benchmarkBufferedBodyRead(b, 1<<10) }
func BenchmarkBufferedBodyRead_1MiB(b *testing.B)  { benchmarkBufferedBodyRead(b, 1<<20) }
func BenchmarkBufferedBodyRead_64MiB(b *testing.B) { benchmarkBufferedBodyRead(b, 64<<20) }

func BenchmarkBufferedBodyRead_1MiB_Parallel(b *testing.B) {
	e := benchHandler(b)
	data := bytes.Repeat([]byte("a"), 1<<20)
	b.SetBytes(int64(len(data)))
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		buf := make([]byte, 32<<10)
		for pb.Next() {
			body := &bufferedBody{handler: e, data: data, reader: bytes.NewReader(data)}
			_, _ = io.CopyBuffer(io.Discard, body, buf)
			_ = body.Close()
		}
	})
}
