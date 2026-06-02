//go:build windows

// System volume control for Windows via WASAPI (Windows Audio Session API).
// Uses COM/IMMDeviceEnumerator → IMMDevice → IAudioEndpointVolume to get and set
// the system master volume without any external tools or CGO dependencies.
package routes

import (
	"fmt"
	"log/slog"
	"syscall"
	"unsafe"

	"github.com/go-ole/go-ole"
)

// WASAPI GUIDs and constants
const (
	// IMMDeviceEnumerator class ID
	clsidMMDeviceEnumerator = "bcde0395-e52f-467c-8e3d-c4579291692e"
	// IMMDeviceEnumerator interface ID
	iidIMMDeviceEnumerator = "a95664d2-9614-4f35-a746-de8db63617e6"
	// IAudioEndpointVolume interface ID
	iidIAudioEndpointVolume = "5cdf2c82-841e-4546-9722-0cf74078229a"

	eRender  = 0
	eConsole = 0

	clsctxInprocServer = 1
)

// Minimal vtable types for IMMDeviceEnumerator and IMMDevice.
// These are plain COM (IUnknown-based), not WinRT.

type wasapiUnknown struct{ vtbl *[256]uintptr }

func (u *wasapiUnknown) release() {
	_, _, _ = syscall.SyscallN(u.vtbl[2], uintptr(unsafe.Pointer(u)))
}

// getDefaultAudioEndpoint calls IMMDeviceEnumerator.GetDefaultAudioEndpoint(eRender, eConsole).
func getDefaultAudioEndpoint(enumerator *wasapiUnknown) (*wasapiUnknown, error) {
	var device *wasapiUnknown
	// vtable slot 4: GetDefaultAudioEndpoint(dataFlow, role, **IMMDevice)
	hr, _, _ := syscall.SyscallN(
		enumerator.vtbl[4],
		uintptr(unsafe.Pointer(enumerator)),
		eRender,
		eConsole,
		uintptr(unsafe.Pointer(&device)),
	)
	if hr != 0 {
		return nil, ole.NewError(hr)
	}
	return device, nil
}

// activateAudioEndpointVolume calls IMMDevice.Activate to get IAudioEndpointVolume.
func activateAudioEndpointVolume(device *wasapiUnknown) (*wasapiUnknown, error) {
	iidVol := ole.NewGUID(iidIAudioEndpointVolume)
	var vol *wasapiUnknown
	// vtable slot 3: Activate(iid, clsctx, pActivationParams, **interface)
	hr, _, _ := syscall.SyscallN(
		device.vtbl[3],
		uintptr(unsafe.Pointer(device)),
		uintptr(unsafe.Pointer(iidVol)),
		clsctxInprocServer,
		0, // pActivationParams = NULL
		uintptr(unsafe.Pointer(&vol)),
	)
	if hr != 0 {
		return nil, ole.NewError(hr)
	}
	return vol, nil
}

// openEndpointVolume initialises COM (idempotent) and returns an IAudioEndpointVolume.
// Caller must call Release() on the returned objects.
func openEndpointVolume() (enumerator, device, vol *wasapiUnknown, err error) {
	// CoInitializeEx is idempotent; S_FALSE means already initialised on this thread.
	if e := ole.CoInitializeEx(0, ole.COINIT_MULTITHREADED); e != nil {
		if oleErr, ok := e.(*ole.OleError); !ok || uintptr(oleErr.Code()) != 0x00000001 {
			return nil, nil, nil, fmt.Errorf("wasapi: CoInitializeEx: %w", e)
		}
	}

	clsid := ole.NewGUID(clsidMMDeviceEnumerator)
	iid := ole.NewGUID(iidIMMDeviceEnumerator)
	unk, e := ole.CreateInstance(clsid, iid)
	if e != nil {
		return nil, nil, nil, fmt.Errorf("wasapi: CreateInstance IMMDeviceEnumerator: %w", e)
	}
	enumerator = (*wasapiUnknown)(unsafe.Pointer(unk))

	device, err = getDefaultAudioEndpoint(enumerator)
	if err != nil {
		enumerator.release()
		return nil, nil, nil, fmt.Errorf("wasapi: GetDefaultAudioEndpoint: %w", err)
	}

	vol, err = activateAudioEndpointVolume(device)
	if err != nil {
		device.release()
		enumerator.release()
		return nil, nil, nil, fmt.Errorf("wasapi: Activate IAudioEndpointVolume: %w", err)
	}
	return enumerator, device, vol, nil
}

// setSystemVolume sets the Windows master volume (0.0–1.0) via WASAPI.
func setSystemVolume(volume float64) error {
	if volume < 0 {
		volume = 0
	}
	if volume > 1 {
		volume = 1
	}

	enumerator, device, vol, err := openEndpointVolume()
	if err != nil {
		slog.Error("wasapi: setSystemVolume failed", "error", err)
		return err
	}
	defer enumerator.release()
	defer device.release()
	defer vol.release()

	// vtable slot 7: SetMasterVolumeLevelScalar(float, *GUID eventContext)
	// We pass a float32 reinterpreted as uintptr, and NULL for eventContext.
	f32 := float32(volume)
	hr, _, _ := syscall.SyscallN(
		vol.vtbl[7],
		uintptr(unsafe.Pointer(vol)),
		uintptr(*(*uint32)(unsafe.Pointer(&f32))), // float32 bit pattern as uint32 → uintptr
		0, // eventContext = NULL
	)
	if hr != 0 {
		return ole.NewError(hr)
	}
	slog.Debug("wasapi: master volume set", "volume", volume)
	return nil
}

// getSystemVolume reads the Windows master volume (0.0–1.0) via WASAPI.
func getSystemVolume() (float64, error) {
	enumerator, device, vol, err := openEndpointVolume()
	if err != nil {
		slog.Error("wasapi: getSystemVolume failed", "error", err)
		return 0.5, err
	}
	defer enumerator.release()
	defer device.release()
	defer vol.release()

	// vtable slot 9: GetMasterVolumeLevelScalar(*float)
	var f32 float32
	hr, _, _ := syscall.SyscallN(
		vol.vtbl[9],
		uintptr(unsafe.Pointer(vol)),
		uintptr(unsafe.Pointer(&f32)),
	)
	if hr != 0 {
		return 0.5, ole.NewError(hr)
	}
	return float64(f32), nil
}



