package cache

// statWriter implements io.Writer and keeps track of the writen bytes.
type statWriter struct {
	written int64
}

func (s *statWriter) Write(p []byte) (n int, err error) {
	size := len(p)
	s.written += int64(size)
	return size, nil
}
