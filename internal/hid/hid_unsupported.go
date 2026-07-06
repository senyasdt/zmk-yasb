//go:build !windows

package hid

import "errors"

func OpenFirst(Filter) (Device, error) {
	return nil, errors.New("HID access is implemented only on Windows")
}

func List() ([]Info, error) {
	return nil, errors.New("HID access is implemented only on Windows")
}
