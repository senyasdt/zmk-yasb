//go:build windows

package hid

import (
	"errors"
	"fmt"
	"syscall"
	"unsafe"
)

const (
	digcfPresent         = 0x00000002
	digcfDeviceInterface = 0x00000010
	genericRead         = 0x80000000
	fileShareRead       = 0x00000001
	fileShareWrite      = 0x00000002
	hidpStatusSuccess   = 0x00110000
	openExisting        = 3
)

var (
	hidDLL     = syscall.NewLazyDLL("hid.dll")
	setupDLL   = syscall.NewLazyDLL("setupapi.dll")
	kernelDLL  = syscall.NewLazyDLL("kernel32.dll")
	procGUID   = hidDLL.NewProc("HidD_GetHidGuid")
	procAttr   = hidDLL.NewProc("HidD_GetAttributes")
	procPrep   = hidDLL.NewProc("HidD_GetPreparsedData")
	procFree   = hidDLL.NewProc("HidD_FreePreparsedData")
	procCaps   = hidDLL.NewProc("HidP_GetCaps")
	procClass  = setupDLL.NewProc("SetupDiGetClassDevsW")
	procEnum   = setupDLL.NewProc("SetupDiEnumDeviceInterfaces")
	procDetail = setupDLL.NewProc("SetupDiGetDeviceInterfaceDetailW")
	procDestroy = setupDLL.NewProc("SetupDiDestroyDeviceInfoList")
	procCreate = kernelDLL.NewProc("CreateFileW")
	procRead   = kernelDLL.NewProc("ReadFile")
	procClose  = kernelDLL.NewProc("CloseHandle")
)

type guid struct {
	Data1 uint32
	Data2 uint16
	Data3 uint16
	Data4 [8]byte
}

type deviceInterfaceData struct {
	Size                uint32
	InterfaceClassGuid  guid
	Flags               uint32
	Reserved            uintptr
}

type hidAttributes struct {
	Size          uint32
	VendorID      uint16
	ProductID     uint16
	VersionNumber uint16
}

type hidCaps struct {
	Usage                     uint16
	UsagePage                 uint16
	InputReportByteLength     uint16
	OutputReportByteLength    uint16
	FeatureReportByteLength   uint16
	Reserved                  [17]uint16
	NumberLinkCollectionNodes uint16
	NumberInputButtonCaps     uint16
	NumberInputValueCaps      uint16
	NumberInputDataIndices    uint16
	NumberOutputButtonCaps    uint16
	NumberOutputValueCaps     uint16
	NumberOutputDataIndices   uint16
	NumberFeatureButtonCaps   uint16
	NumberFeatureValueCaps    uint16
	NumberFeatureDataIndices  uint16
}

type windowsDevice struct {
	path       string
	handle     uintptr
	inputBytes uint16
}

func OpenFirst(filter Filter) (Device, error) {
	paths, err := enumeratePaths()
	if err != nil {
		return nil, err
	}
	for _, path := range paths {
		dev, err := openPath(path)
		if err != nil {
			continue
		}
		ok, err := dev.matches(filter)
		if err == nil && ok {
			return dev, nil
		}
		_ = dev.Close()
	}
	return nil, fmt.Errorf("no HID device matched VID=%04X PID=%04X usage_page=%04X usage=%04X", filter.VID, filter.PID, filter.UsagePage, filter.Usage)
}

func List() ([]Info, error) {
	paths, err := enumeratePaths()
	if err != nil {
		return nil, err
	}
	var infos []Info
	for _, path := range paths {
		dev, err := openPath(path)
		if err != nil {
			continue
		}
		info, err := dev.info()
		_ = dev.Close()
		if err == nil {
			infos = append(infos, info)
		}
	}
	return infos, nil
}

func (d *windowsDevice) Path() string {
	return d.path
}

func (d *windowsDevice) InputReportBytes() uint16 {
	return d.inputBytes
}

func (d *windowsDevice) Close() error {
	if d.handle == 0 || d.handle == ^uintptr(0) {
		return nil
	}
	r, _, err := procClose.Call(d.handle)
	if r == 0 {
		return err
	}
	d.handle = 0
	return nil
}

func (d *windowsDevice) Read(buf []byte) (int, error) {
	var read uint32
	r, _, err := procRead.Call(
		d.handle,
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(uint32(len(buf))),
		uintptr(unsafe.Pointer(&read)),
		0,
	)
	if r == 0 {
		if err != syscall.Errno(0) {
			return 0, err
		}
		return 0, errors.New("ReadFile failed")
	}
	return int(read), nil
}

func (d *windowsDevice) matches(filter Filter) (bool, error) {
	info, err := d.info()
	if err != nil {
		return false, err
	}
	d.inputBytes = info.InputBytes
	return info.VID == filter.VID && info.PID == filter.PID && info.UsagePage == filter.UsagePage && info.Usage == filter.Usage, nil
}

func (d *windowsDevice) info() (Info, error) {
	attr := hidAttributes{Size: uint32(unsafe.Sizeof(hidAttributes{}))}
	r, _, err := procAttr.Call(d.handle, uintptr(unsafe.Pointer(&attr)))
	if r == 0 {
		return Info{}, err
	}

	var prep uintptr
	r, _, err = procPrep.Call(d.handle, uintptr(unsafe.Pointer(&prep)))
	if r == 0 {
		return Info{}, err
	}
	defer procFree.Call(prep)

	var caps hidCaps
	r, _, err = procCaps.Call(prep, uintptr(unsafe.Pointer(&caps)))
	if r != hidpStatusSuccess {
		return Info{}, err
	}
	return Info{
		Path:       d.path,
		VID:        attr.VendorID,
		PID:        attr.ProductID,
		UsagePage:  caps.UsagePage,
		Usage:      caps.Usage,
		InputBytes: caps.InputReportByteLength,
	}, nil
}

func enumeratePaths() ([]string, error) {
	var hidGuid guid
	procGUID.Call(uintptr(unsafe.Pointer(&hidGuid)))

	hInfo, _, err := procClass.Call(
		uintptr(unsafe.Pointer(&hidGuid)),
		0,
		0,
		digcfPresent|digcfDeviceInterface,
	)
	if hInfo == ^uintptr(0) {
		return nil, err
	}
	defer procDestroy.Call(hInfo)

	var paths []string
	for index := uint32(0); ; index++ {
		iface := deviceInterfaceData{Size: uint32(unsafe.Sizeof(deviceInterfaceData{}))}
		r, _, _ := procEnum.Call(
			hInfo,
			0,
			uintptr(unsafe.Pointer(&hidGuid)),
			uintptr(index),
			uintptr(unsafe.Pointer(&iface)),
		)
		if r == 0 {
			break
		}

		var needed uint32
		procDetail.Call(hInfo, uintptr(unsafe.Pointer(&iface)), 0, 0, uintptr(unsafe.Pointer(&needed)), 0)
		if needed == 0 {
			continue
		}
		buf := make([]byte, needed)
		size := uint32(8)
		if unsafe.Sizeof(uintptr(0)) == 4 {
			size = 6
		}
		*(*uint32)(unsafe.Pointer(&buf[0])) = size
		r, _, _ = procDetail.Call(
			hInfo,
			uintptr(unsafe.Pointer(&iface)),
			uintptr(unsafe.Pointer(&buf[0])),
			uintptr(needed),
			uintptr(unsafe.Pointer(&needed)),
			0,
		)
		if r == 0 {
			continue
		}
		pathOffset := uint32(4)
		u16 := unsafe.Slice((*uint16)(unsafe.Pointer(&buf[pathOffset])), (len(buf)-int(pathOffset))/2)
		paths = append(paths, syscall.UTF16ToString(u16))
	}
	return paths, nil
}

func openPath(path string) (*windowsDevice, error) {
	ptr, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return nil, err
	}
	h, _, callErr := procCreate.Call(
		uintptr(unsafe.Pointer(ptr)),
		genericRead,
		fileShareRead|fileShareWrite,
		0,
		openExisting,
		0,
		0,
	)
	if h == ^uintptr(0) {
		return nil, callErr
	}
	return &windowsDevice{path: path, handle: h}, nil
}
