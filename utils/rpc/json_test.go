// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package rpc

import (
	"bytes"
	"io"
	"strings"
	"testing"
)

type mockReadCloser struct {
	reader  io.Reader
	closed  bool
	readAll bool
}

func (m *mockReadCloser) Read(p []byte) (n int, err error) {
	n, err = m.reader.Read(p)
	if err == io.EOF {
		m.readAll = true
	}
	return n, err
}

func (m *mockReadCloser) Close() error {
	m.closed = true
	return nil
}

func TestCleanlyCloseBody_NilBody(t *testing.T) {
	err := CleanlyCloseBody(nil)
	if err != nil {
		t.Errorf("CleanlyCloseBody(nil) returned error: %v", err)
	}
}

func TestCleanlyCloseBody_EmptyBody(t *testing.T) {
	mock := &mockReadCloser{
		reader: bytes.NewReader([]byte{}),
	}

	err := CleanlyCloseBody(mock)
	if err != nil {
		t.Errorf("CleanlyCloseBody() returned error: %v", err)
	}

	if !mock.closed {
		t.Error("Body was not closed")
	}

	if !mock.readAll {
		t.Error("Body was not fully read")
	}
}

func TestCleanlyCloseBody_WithData(t *testing.T) {
	testData := "This is test data that should be drained"
	mock := &mockReadCloser{
		reader: strings.NewReader(testData),
	}

	err := CleanlyCloseBody(mock)
	if err != nil {
		t.Errorf("CleanlyCloseBody() returned error: %v", err)
	}

	if !mock.closed {
		t.Error("Body was not closed")
	}

	if !mock.readAll {
		t.Error("Body was not fully read (not drained)")
	}
}

func TestCleanlyCloseBody_LargeBody(t *testing.T) {
	// Create a large body (1MB)
	largeData := bytes.Repeat([]byte("x"), 1024*1024)
	mock := &mockReadCloser{
		reader: bytes.NewReader(largeData),
	}

	err := CleanlyCloseBody(mock)
	if err != nil {
		t.Errorf("CleanlyCloseBody() returned error: %v", err)
	}

	if !mock.closed {
		t.Error("Body was not closed")
	}

	if !mock.readAll {
		t.Error("Large body was not fully drained")
	}
}

func TestCleanlyCloseBody_PartiallyReadBody(t *testing.T) {
	testData := "This is test data"
	mock := &mockReadCloser{
		reader: strings.NewReader(testData),
	}

	// Partially read the body
	buf := make([]byte, 4)
	_, err := mock.Read(buf)
	if err != nil {
		t.Fatalf("Failed to partially read: %v", err)
	}

	// Verify not fully read yet
	if mock.readAll {
		t.Error("Body should not be fully read yet")
	}

	// Now cleanly close should drain the rest
	err = CleanlyCloseBody(mock)
	if err != nil {
		t.Errorf("CleanlyCloseBody() returned error: %v", err)
	}

	if !mock.closed {
		t.Error("Body was not closed")
	}

	if !mock.readAll {
		t.Error("Remaining body data was not drained")
	}
}

func BenchmarkCleanlyCloseBody_Small(b *testing.B) {
	data := []byte("small response body")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mock := &mockReadCloser{
			reader: bytes.NewReader(data),
		}
		CleanlyCloseBody(mock)
	}
}

func BenchmarkCleanlyCloseBody_Large(b *testing.B) {
	// 1MB response
	data := bytes.Repeat([]byte("x"), 1024*1024)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mock := &mockReadCloser{
			reader: bytes.NewReader(data),
		}
		CleanlyCloseBody(mock)
	}
}
