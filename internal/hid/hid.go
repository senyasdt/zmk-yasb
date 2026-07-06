package hid

type Filter struct {
	VID       uint16
	PID       uint16
	UsagePage uint16
	Usage     uint16
}

type Device interface {
	Path() string
	InputReportBytes() uint16
	Read([]byte) (int, error)
	Close() error
}

type Info struct {
	Path       string
	VID        uint16
	PID        uint16
	UsagePage  uint16
	Usage      uint16
	InputBytes uint16
}
