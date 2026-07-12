package alert

import "bytes"

// readerOf 把字节切片包装成 *bytes.Reader, 复用 bytes.NewReader 的 Read/Seek 实现。
func readerOf(b []byte) *bytes.Reader {
	return bytes.NewReader(b)
}
