package audio

/*
#cgo LDFLAGS: -framework CoreAudio -framework CoreFoundation
#include <CoreAudio/CoreAudio.h>
#include <CoreFoundation/CoreFoundation.h>
#include <stdlib.h>
#include <string.h>

static AudioDeviceID getDefaultOutputDevice() {
	AudioDeviceID deviceID = 0;
	UInt32 size = sizeof(AudioDeviceID);
	AudioObjectPropertyAddress addr = {
		kAudioHardwarePropertyDefaultOutputDevice,
		kAudioObjectPropertyScopeGlobal,
		0 // kAudioObjectPropertyElementMain
	};
	AudioObjectGetPropertyData(kAudioObjectSystemObject, &addr, 0, NULL, &size, &deviceID);
	return deviceID;
}

static OSStatus setDefaultOutputDevice(AudioDeviceID deviceID) {
	AudioObjectPropertyAddress addr = {
		kAudioHardwarePropertyDefaultOutputDevice,
		kAudioObjectPropertyScopeGlobal,
		0
	};
	return AudioObjectSetPropertyData(kAudioObjectSystemObject, &addr, 0, NULL, sizeof(AudioDeviceID), &deviceID);
}

static UInt32 getDeviceCount() {
	AudioObjectPropertyAddress addr = {
		kAudioHardwarePropertyDevices,
		kAudioObjectPropertyScopeGlobal,
		0
	};
	UInt32 size = 0;
	AudioObjectGetPropertyDataSize(kAudioObjectSystemObject, &addr, 0, NULL, &size);
	return size / sizeof(AudioDeviceID);
}

static void getDeviceIDs(AudioDeviceID *devices, UInt32 count) {
	AudioObjectPropertyAddress addr = {
		kAudioHardwarePropertyDevices,
		kAudioObjectPropertyScopeGlobal,
		0
	};
	UInt32 size = count * sizeof(AudioDeviceID);
	AudioObjectGetPropertyData(kAudioObjectSystemObject, &addr, 0, NULL, &size, devices);
}

// Returns a malloc'd C string. Caller must free.
static char* getDeviceName(AudioDeviceID deviceID) {
	AudioObjectPropertyAddress addr = {
		kAudioObjectPropertyName,
		kAudioObjectPropertyScopeGlobal,
		0
	};
	CFStringRef name = NULL;
	UInt32 size = sizeof(CFStringRef);
	OSStatus status = AudioObjectGetPropertyData(deviceID, &addr, 0, NULL, &size, &name);
	if (status != 0 || name == NULL) {
		return NULL;
	}
	CFIndex len = CFStringGetMaximumSizeForEncoding(CFStringGetLength(name), kCFStringEncodingUTF8) + 1;
	char *buf = (char *)malloc(len);
	CFStringGetCString(name, buf, len, kCFStringEncodingUTF8);
	CFRelease(name);
	return buf;
}
*/
import "C"
import (
	"fmt"
	"strings"
	"unsafe"
)

// CoreAudioDeviceID wraps the macOS AudioDeviceID.
type CoreAudioDeviceID = C.AudioDeviceID

// GetDefaultOutputDeviceID returns the current system default output device ID.
func GetDefaultOutputDeviceID() CoreAudioDeviceID {
	return C.getDefaultOutputDevice()
}

// SetDefaultOutputDevice sets the system default output to the given device ID.
func SetDefaultOutputDevice(id CoreAudioDeviceID) error {
	status := C.setDefaultOutputDevice(id)
	if status != 0 {
		return fmt.Errorf("failed to set default output device (OSStatus %d)", status)
	}
	return nil
}

// FindCoreAudioDevice finds a CoreAudio device ID by name substring match.
func FindCoreAudioDevice(nameSubstr string) (CoreAudioDeviceID, string, error) {
	count := C.getDeviceCount()
	if count == 0 {
		return 0, "", fmt.Errorf("no audio devices found")
	}

	ids := make([]C.AudioDeviceID, count)
	C.getDeviceIDs(&ids[0], count)

	needle := strings.ToLower(nameSubstr)
	for _, id := range ids {
		cName := C.getDeviceName(id)
		if cName == nil {
			continue
		}
		name := C.GoString(cName)
		C.free(unsafe.Pointer(cName))

		if strings.Contains(strings.ToLower(name), needle) {
			return id, name, nil
		}
	}
	return 0, "", fmt.Errorf("no CoreAudio device matching %q found", nameSubstr)
}

// GetDeviceName returns the name of a CoreAudio device by ID.
func GetDeviceName(id CoreAudioDeviceID) string {
	cName := C.getDeviceName(id)
	if cName == nil {
		return "<unknown>"
	}
	name := C.GoString(cName)
	C.free(unsafe.Pointer(cName))
	return name
}
