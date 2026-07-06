package hid

type Filter struct {
	VID       uint16
	PID       uint16
	UsagePage uint16
	Usage     uint16
}

type Device interface {
	Path() string
	Read([]byte) (int, error)
	Close() error
}
