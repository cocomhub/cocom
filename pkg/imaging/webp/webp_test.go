// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package webp

import (
	"bytes"
	"encoding/binary"
	"errors"
	"testing"
)

// writeRIFFChunk 按 RIFF 规范写入一个 chunk（奇数长度补 0）。
func writeRIFFChunk(w *bytes.Buffer, id string, payload []byte) {
	w.WriteString(id)
	var size [4]byte
	binary.LittleEndian.PutUint32(size[:], uint32(len(payload)))
	w.Write(size[:])
	w.Write(payload)
	if len(payload)%2 == 1 {
		w.WriteByte(0)
	}
}

// buildTruncatedAlphaWebP 构造「VP8X 声明带 alpha + ALPH chunk 无数据」的 WebP。
// ALPH 预处理字节缺失会令 io.ReadFull 失败；此处若把该错误吞掉就会返回 (nil, nil)。
func buildTruncatedAlphaWebP() []byte {
	var vp8x [10]byte
	vp8x[0] = 1 << 4 // alphaBit；width-1 = height-1 = 0（1x1）

	var body bytes.Buffer
	body.WriteString("WEBP")
	writeRIFFChunk(&body, "VP8X", vp8x[:])
	writeRIFFChunk(&body, "ALPH", nil)

	var out bytes.Buffer
	out.WriteString("RIFF")
	var size [4]byte
	binary.LittleEndian.PutUint32(size[:], uint32(body.Len()))
	out.Write(size[:])
	out.Write(body.Bytes())
	return out.Bytes()
}

func TestDecode_TruncatedAlphaChunkReturnsError(t *testing.T) {
	img, err := Decode(bytes.NewReader(buildTruncatedAlphaWebP()))
	if err == nil {
		t.Fatalf("Decode(ALPH 数据缺失) = (%v, nil)；必须返回错误而不是 (nil, nil)", img)
	}
	if img != nil {
		t.Errorf("Decode(ALPH 数据缺失) 同时返回了图像 %v 与非 nil 错误", img)
	}
}

func TestDecode_InvalidDataReturnsError(t *testing.T) {
	img, err := Decode(bytes.NewReader([]byte{0x00}))
	if err == nil {
		t.Fatalf("Decode(非法数据) = (%v, nil)；必须返回错误", img)
	}
}

func TestReadAlpha_Uncompressed(t *testing.T) {
	// widthMinusOne=1 / heightMinusOne=1 → 2x2，压缩方式 0（原样）。
	data := []byte{1, 2, 3, 4}
	alpha, stride, err := readAlpha(bytes.NewReader(data), 1, 1, 0)
	if err != nil {
		t.Fatalf("readAlpha: %v", err)
	}
	if stride != 2 {
		t.Errorf("stride = %d, want 2", stride)
	}
	if !bytes.Equal(alpha, data) {
		t.Errorf("alpha = %v, want %v", alpha, data)
	}
}

func TestReadAlpha_ShortDataReturnsError(t *testing.T) {
	// 2x2 需要 4 字节，只给 2 字节。
	if _, _, err := readAlpha(bytes.NewReader([]byte{1, 2}), 1, 1, 0); err == nil {
		t.Fatal("readAlpha(数据不足) 必须返回错误")
	}
}

func TestReadAlpha_RejectsDecompressionBomb(t *testing.T) {
	// 100001x100001 远超 maxAlphaPixels（1e8），必须在分配内存前拒绝。
	_, _, err := readAlpha(bytes.NewReader(nil), 100000, 100000, 0)
	if err == nil {
		t.Fatal("readAlpha(超大尺寸) 必须返回错误")
	}
	if !errors.Is(err, errInvalidFormat) {
		t.Errorf("err = %v, want errors.Is(err, errInvalidFormat)", err)
	}
}

func TestReadAlpha_UnsupportedCompression(t *testing.T) {
	_, _, err := readAlpha(bytes.NewReader(nil), 1, 1, 9)
	if err == nil {
		t.Fatal("readAlpha(未知压缩方式) 必须返回错误")
	}
	if !errors.Is(err, errInvalidFormat) {
		t.Errorf("err = %v, want errors.Is(err, errInvalidFormat)", err)
	}
}
