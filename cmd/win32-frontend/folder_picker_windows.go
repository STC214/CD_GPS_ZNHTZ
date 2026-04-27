package main

import (
	"errors"
	"fmt"
	"strings"
	"syscall"
	"unsafe"
)

const (
	COINIT_APARTMENTTHREADED = 0x2
	COINIT_DISABLE_OLE1DDE   = 0x4
	CLSCTX_INPROC_SERVER     = 0x1
	FOS_FORCEFILESYSTEM      = 0x00000040
	FOS_PICKFOLDERS          = 0x00000020
	FOS_PATHMUSTEXIST        = 0x00000800
	SIGDN_FILESYSPATH        = 0x80058000
)

var (
	procCoInitializeEx            = ole32.NewProc("CoInitializeEx")
	procCoUninitialize            = ole32.NewProc("CoUninitialize")
	procCoCreateInstance          = ole32.NewProc("CoCreateInstance")
	procSHCreateItemFromParsingName = shell32.NewProc("SHCreateItemFromParsingName")
)

var errFolderDialogCanceled = errors.New("folder dialog canceled")

type GUID struct {
	Data1 uint32
	Data2 uint16
	Data3 uint16
	Data4 [8]byte
}

type IFileDialog struct {
	lpVtbl *iFileDialogVtbl
}

type iFileDialogVtbl struct {
	QueryInterface      uintptr
	AddRef              uintptr
	Release             uintptr
	Show                uintptr
	SetFileTypes        uintptr
	SetFileTypeIndex    uintptr
	GetFileTypeIndex    uintptr
	Advise              uintptr
	Unadvise            uintptr
	SetOptions          uintptr
	GetOptions          uintptr
	SetDefaultFolder    uintptr
	SetFolder           uintptr
	GetFolder           uintptr
	GetCurrentSelection uintptr
	SetFileName         uintptr
	GetFileName         uintptr
	SetTitle            uintptr
	SetOkButtonLabel    uintptr
	SetFileNameLabel    uintptr
	GetResult           uintptr
	AddPlace            uintptr
	SetDefaultExtension uintptr
	Close               uintptr
	SetClientGuid       uintptr
	ClearClientData     uintptr
	SetFilter           uintptr
}

type IShellItem struct {
	lpVtbl *iShellItemVtbl
}

type iShellItemVtbl struct {
	QueryInterface uintptr
	AddRef         uintptr
	Release        uintptr
	BindToHandler  uintptr
	GetParent      uintptr
	GetDisplayName  uintptr
	Compare        uintptr
}

var (
	clsidFileOpenDialog = GUID{0xDC1C5A9C, 0xE88A, 0x4DDE, [8]byte{0xA5, 0xA1, 0x60, 0xF8, 0x2A, 0x20, 0xAE, 0xF7}}
	iidIFileDialog      = GUID{0x42F85136, 0xDB7E, 0x439C, [8]byte{0x85, 0xF1, 0xE4, 0x07, 0x5D, 0x13, 0x5F, 0xC8}}
	iidIShellItem       = GUID{0x43826D1E, 0xE718, 0x42EE, [8]byte{0xBC, 0x55, 0xA1, 0xE2, 0x61, 0xC3, 0x7B, 0xFE}}
)

func browseFolderDialog(hwnd HWND, title, initialPath string) (string, error) {
	path, err := browseFolderDialogExplorer(hwnd, title, initialPath)
	if errors.Is(err, errFolderDialogCanceled) {
		return "", nil
	}
	return path, err
}

func browseFolderDialogExplorer(hwnd HWND, title, initialPath string) (string, error) {
	release, err := coInitializeDialogApartment()
	if err != nil {
		return "", err
	}
	defer release()

	dialog, err := createFileDialog()
	if err != nil {
		return "", err
	}
	defer dialog.release()

	if err := dialog.setOptions(FOS_PICKFOLDERS | FOS_FORCEFILESYSTEM | FOS_PATHMUSTEXIST); err != nil {
		return "", err
	}
	if strings.TrimSpace(title) != "" {
		if err := dialog.setTitle(title); err != nil {
			return "", err
		}
	}
	if folder := strings.TrimSpace(initialPath); folder != "" {
		if item, err := createShellItem(folder); err == nil && item != nil {
			_ = dialog.setDefaultFolder(item)
			item.release()
		}
	}
	if err := dialog.show(hwnd); err != nil {
		if errors.Is(err, errFolderDialogCanceled) {
			return "", nil
		}
		return "", err
	}
	item, err := dialog.getResult()
	if err != nil {
		if errors.Is(err, errFolderDialogCanceled) {
			return "", nil
		}
		return "", err
	}
	defer item.release()
	return item.displayName(SIGDN_FILESYSPATH)
}

func coInitializeDialogApartment() (func() error, error) {
	hr, _, _ := procCoInitializeEx.Call(0, uintptr(COINIT_APARTMENTTHREADED|COINIT_DISABLE_OLE1DDE))
	switch hr {
	case 0, 1:
		return func() error {
			procCoUninitialize.Call()
			return nil
		}, nil
	default:
		return nil, hresultError(hr)
	}
}

func createFileDialog() (*IFileDialog, error) {
	var dialog *IFileDialog
	hr, _, _ := procCoCreateInstance.Call(
		uintptr(unsafe.Pointer(&clsidFileOpenDialog)),
		0,
		uintptr(CLSCTX_INPROC_SERVER),
		uintptr(unsafe.Pointer(&iidIFileDialog)),
		uintptr(unsafe.Pointer(&dialog)),
	)
	if hr != 0 {
		return nil, hresultError(hr)
	}
	if dialog == nil {
		return nil, fmt.Errorf("create file dialog: nil dialog")
	}
	return dialog, nil
}

func createShellItem(path string) (*IShellItem, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, nil
	}
	var item *IShellItem
	pathPtr, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return nil, err
	}
	hr, _, _ := procSHCreateItemFromParsingName.Call(
		uintptr(unsafe.Pointer(pathPtr)),
		0,
		uintptr(unsafe.Pointer(&iidIShellItem)),
		uintptr(unsafe.Pointer(&item)),
	)
	if hr != 0 {
		return nil, hresultError(hr)
	}
	return item, nil
}

func hresultError(hr uintptr) error {
	if hr == 0 || hr == 1 {
		return nil
	}
	if hr == 0x800704C7 {
		return errFolderDialogCanceled
	}
	return syscall.Errno(hr)
}

func (d *IFileDialog) release() {
	if d != nil && d.lpVtbl != nil && d.lpVtbl.Release != 0 {
		syscall.SyscallN(d.lpVtbl.Release, uintptr(unsafe.Pointer(d)))
	}
}

func (d *IFileDialog) setOptions(options uint32) error {
	hr, _, _ := syscall.SyscallN(d.lpVtbl.SetOptions, uintptr(unsafe.Pointer(d)), uintptr(options))
	return hresultError(hr)
}

func (d *IFileDialog) setTitle(title string) error {
	titlePtr, err := syscall.UTF16PtrFromString(title)
	if err != nil {
		return err
	}
	hr, _, _ := syscall.SyscallN(d.lpVtbl.SetTitle, uintptr(unsafe.Pointer(d)), uintptr(unsafe.Pointer(titlePtr)))
	return hresultError(hr)
}

func (d *IFileDialog) setDefaultFolder(item *IShellItem) error {
	if item == nil {
		return nil
	}
	hr, _, _ := syscall.SyscallN(d.lpVtbl.SetDefaultFolder, uintptr(unsafe.Pointer(d)), uintptr(unsafe.Pointer(item)))
	return hresultError(hr)
}

func (d *IFileDialog) show(hwnd HWND) error {
	hr, _, _ := syscall.SyscallN(d.lpVtbl.Show, uintptr(unsafe.Pointer(d)), uintptr(hwnd))
	return hresultError(hr)
}

func (d *IFileDialog) getResult() (*IShellItem, error) {
	var item *IShellItem
	hr, _, _ := syscall.SyscallN(d.lpVtbl.GetResult, uintptr(unsafe.Pointer(d)), uintptr(unsafe.Pointer(&item)))
	if hr != 0 {
		return nil, hresultError(hr)
	}
	return item, nil
}

func (s *IShellItem) release() {
	if s != nil && s.lpVtbl != nil && s.lpVtbl.Release != 0 {
		syscall.SyscallN(s.lpVtbl.Release, uintptr(unsafe.Pointer(s)))
	}
}

func (s *IShellItem) displayName(sigdnName uint32) (string, error) {
	if s == nil || s.lpVtbl == nil {
		return "", fmt.Errorf("shell item is nil")
	}
	var name *uint16
	hr, _, _ := syscall.SyscallN(s.lpVtbl.GetDisplayName, uintptr(unsafe.Pointer(s)), uintptr(sigdnName), uintptr(unsafe.Pointer(&name)))
	if hr != 0 {
		return "", hresultError(hr)
	}
	if name == nil {
		return "", nil
	}
	defer procCoTaskMemFree.Call(uintptr(unsafe.Pointer(name)))
	return syscall.UTF16ToString((*[1 << 20]uint16)(unsafe.Pointer(name))[:]), nil
}
