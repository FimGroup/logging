package logging

type nilWriteCloser struct {
}

func (n2 nilWriteCloser) Write(p []byte) (n int, err error) {
	return
}

func (n2 nilWriteCloser) Close() error {
	return nil
}
