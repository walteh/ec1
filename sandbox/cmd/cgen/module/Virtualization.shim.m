#import <Foundation/Foundation.h>
#import <objc/message.h>
#import <objc/runtime.h>
#import <Virtualization/Virtualization.h>

#ifdef __cplusplus
extern "C" {
#endif

// Shim for Swift property getter: minimumSupportedCPUCount
void* c_objc_cs_VZMacOSConfigurationRequirements_py_minimumSupportedCPUCount_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("minimumSupportedCPUCount"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: minimumSupportedCPUCount
void c_objc_cs_VZMacOSConfigurationRequirements_py_minimumSupportedCPUCount_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setMinimumSupportedCPUCount:"), val);
}

// Shim for Swift property getter: isReadOnly
void* c_objc_cs_VZDiskImageStorageDeviceAttachment_py_readOnly_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("isReadOnly"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: isReadOnly
void c_objc_cs_VZDiskImageStorageDeviceAttachment_py_readOnly_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setIsReadOnly:"), val);
}

// Shim for Swift property getter: usbDevices
void* c_objc_cs_VZUSBController_py_usbDevices_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("usbDevices"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: usbDevices
void c_objc_cs_VZUSBController_py_usbDevices_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setUsbDevices:"), val);
}

// Shim for Swift property getter: isReadOnly
void* c_objc_cs_VZDiskBlockDeviceStorageDeviceAttachment_py_readOnly_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("isReadOnly"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: isReadOnly
void c_objc_cs_VZDiskBlockDeviceStorageDeviceAttachment_py_readOnly_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setIsReadOnly:"), val);
}

// Shim for Swift property getter: consoleDevices
void* c_objc_cs_VZVirtualMachineConfiguration_py_consoleDevices_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("consoleDevices"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: consoleDevices
void c_objc_cs_VZVirtualMachineConfiguration_py_consoleDevices_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setConsoleDevices:"), val);
}

// Shim for Swift property getter: share
void* c_objc_cs_VZVirtioFileSystemDevice_py_share_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("share"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: share
void c_objc_cs_VZVirtioFileSystemDevice_py_share_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setShare:"), val);
}

// Shim for Swift property getter: fileHandle
void* c_objc_cs_VZDiskBlockDeviceStorageDeviceAttachment_py_fileHandle_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("fileHandle"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: fileHandle
void c_objc_cs_VZDiskBlockDeviceStorageDeviceAttachment_py_fileHandle_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setFileHandle:"), val);
}

// Shim for Swift property getter: uuid
void* c_objc_pl_VZUSBDevice_py_uuid_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("uuid"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: uuid
void c_objc_pl_VZUSBDevice_py_uuid_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setUuid:"), val);
}

// Shim for Swift property getter: minimumSupportedMemorySize
void* c_objc_cs_VZMacOSConfigurationRequirements_py_minimumSupportedMemorySize_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("minimumSupportedMemorySize"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: minimumSupportedMemorySize
void c_objc_cs_VZMacOSConfigurationRequirements_py_minimumSupportedMemorySize_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setMinimumSupportedMemorySize:"), val);
}

// Shim for Swift property getter: scanouts
void* c_objc_cs_VZVirtioGraphicsDeviceConfiguration_py_scanouts_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("scanouts"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: scanouts
void c_objc_cs_VZVirtioGraphicsDeviceConfiguration_py_scanouts_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setScanouts:"), val);
}

// Shim for Swift property getter: initialRamdiskURL
void* c_objc_cs_VZLinuxBootLoader_py_initialRamdiskURL_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("initialRamdiskURL"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: initialRamdiskURL
void c_objc_cs_VZLinuxBootLoader_py_initialRamdiskURL_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setInitialRamdiskURL:"), val);
}

// Shim for Swift property getter: canRequestStop
void* c_objc_cs_VZVirtualMachine_py_canRequestStop_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("canRequestStop"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: canRequestStop
void c_objc_cs_VZVirtualMachine_py_canRequestStop_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setCanRequestStop:"), val);
}

// Shim for Swift property getter: delegate
void* c_objc_cs_VZVirtioConsoleDevice_py_delegate_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("delegate"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: delegate
void c_objc_cs_VZVirtioConsoleDevice_py_delegate_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setDelegate:"), val);
}

// Shim for Swift property getter: url
void* c_objc_cs_VZNetworkBlockDeviceStorageDeviceAttachment_py_URL_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("url"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: url
void c_objc_cs_VZNetworkBlockDeviceStorageDeviceAttachment_py_URL_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setUrl:"), val);
}

// Shim for Swift property getter: synchronizationMode
void* c_objc_cs_VZDiskBlockDeviceStorageDeviceAttachment_py_synchronizationMode_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("synchronizationMode"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: synchronizationMode
void c_objc_cs_VZDiskBlockDeviceStorageDeviceAttachment_py_synchronizationMode_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setSynchronizationMode:"), val);
}

// Shim for Swift property getter: canResume
void* c_objc_cs_VZVirtualMachine_py_canResume_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("canResume"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: canResume
void c_objc_cs_VZVirtualMachine_py_canResume_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setCanResume:"), val);
}

// Shim for Swift property getter: serialPorts
void* c_objc_cs_VZVirtualMachineConfiguration_py_serialPorts_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("serialPorts"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: serialPorts
void c_objc_cs_VZVirtualMachineConfiguration_py_serialPorts_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setSerialPorts:"), val);
}

// Shim for Swift property getter: cachingMode
void* c_objc_cs_VZDiskImageStorageDeviceAttachment_py_cachingMode_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("cachingMode"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: cachingMode
void c_objc_cs_VZDiskImageStorageDeviceAttachment_py_cachingMode_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setCachingMode:"), val);
}

// Shim for Swift property getter: fileHandleForWriting
void* c_objc_cs_VZFileHandleSerialPortAttachment_py_fileHandleForWriting_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("fileHandleForWriting"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: fileHandleForWriting
void c_objc_cs_VZFileHandleSerialPortAttachment_py_fileHandleForWriting_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setFileHandleForWriting:"), val);
}

// Shim for Swift property getter: canPause
void* c_objc_cs_VZVirtualMachine_py_canPause_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("canPause"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: canPause
void c_objc_cs_VZVirtualMachine_py_canPause_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setCanPause:"), val);
}

// Shim for Swift property getter: hardwareModel
void* c_objc_cs_VZMacOSConfigurationRequirements_py_hardwareModel_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("hardwareModel"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: hardwareModel
void c_objc_cs_VZMacOSConfigurationRequirements_py_hardwareModel_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setHardwareModel:"), val);
}

// Shim for Swift property getter: isNestedVirtualizationEnabled
void* c_objc_cs_VZGenericPlatformConfiguration_py_nestedVirtualizationEnabled_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("isNestedVirtualizationEnabled"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: isNestedVirtualizationEnabled
void c_objc_cs_VZGenericPlatformConfiguration_py_nestedVirtualizationEnabled_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setIsNestedVirtualizationEnabled:"), val);
}

// Shim for Swift method: resume()
void* c_objc_cs_VZVirtualMachine_im_resumeWithCompletionHandler_(void* self) {
    id obj = (__bridge id)self;
    typedef void* (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("resume()"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property getter: attachment
void* c_objc_cs_VZStorageDeviceConfiguration_py_attachment_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("attachment"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: attachment
void c_objc_cs_VZStorageDeviceConfiguration_py_attachment_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setAttachment:"), val);
}

// Shim for Swift method: detach(device:)
void* c_objc_cs_VZUSBController_im_detachDevice_completionHandler_(void* self, void* device) {
    id obj = (__bridge id)self;
    typedef void* (*MsgFn)(id, SEL, id, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj,
               sel_getUid("detach(device:):"),
               device);
    return (__bridge_retained void*)rv;
}

// Shim for Swift property getter: widthInPixels
void* c_objc_cs_VZVirtioGraphicsScanoutConfiguration_py_widthInPixels_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("widthInPixels"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: widthInPixels
void c_objc_cs_VZVirtioGraphicsScanoutConfiguration_py_widthInPixels_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setWidthInPixels:"), val);
}

// Shim for Swift method: validateSaveRestoreSupport()
void* c_objc_cs_VZVirtualMachineConfiguration_im_validateSaveRestoreSupportWithError_(void* self) {
    id obj = (__bridge id)self;
    typedef void* (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("validateSaveRestoreSupport()"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property getter: directorySharingDevices
void* c_objc_cs_VZVirtualMachineConfiguration_py_directorySharingDevices_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("directorySharingDevices"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: directorySharingDevices
void c_objc_cs_VZVirtualMachineConfiguration_py_directorySharingDevices_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setDirectorySharingDevices:"), val);
}

// Shim for Swift property getter: timeout
void* c_objc_cs_VZNetworkBlockDeviceStorageDeviceAttachment_py_timeout_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("timeout"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: timeout
void c_objc_cs_VZNetworkBlockDeviceStorageDeviceAttachment_py_timeout_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setTimeout:"), val);
}

// Shim for Swift property getter: url
void* c_objc_cs_VZDiskImageStorageDeviceAttachment_py_URL_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("url"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: url
void c_objc_cs_VZDiskImageStorageDeviceAttachment_py_URL_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setUrl:"), val);
}

// Shim for Swift method: validate()
void* c_objc_cs_VZVirtualMachineConfiguration_im_validateWithError_(void* self) {
    id obj = (__bridge id)self;
    typedef void* (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("validate()"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property getter: directorySharingDevices
void* c_objc_cs_VZVirtualMachine_py_directorySharingDevices_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("directorySharingDevices"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: directorySharingDevices
void c_objc_cs_VZVirtualMachine_py_directorySharingDevices_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setDirectorySharingDevices:"), val);
}

// Shim for Swift method: connect(toPort:)
void* c_objc_cs_VZVirtioSocketDevice_im_connectToPort_completionHandler_(void* self, void* port) {
    id obj = (__bridge id)self;
    typedef void* (*MsgFn)(id, SEL, id, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj,
               sel_getUid("connect(toPort:):"),
               port);
    return (__bridge_retained void*)rv;
}

// Shim for Swift property getter: memoryBalloonDevices
void* c_objc_cs_VZVirtualMachineConfiguration_py_memoryBalloonDevices_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("memoryBalloonDevices"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: memoryBalloonDevices
void c_objc_cs_VZVirtualMachineConfiguration_py_memoryBalloonDevices_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setMemoryBalloonDevices:"), val);
}

// Shim for Swift property getter: uuid
void* c_objc_pl_VZUSBDeviceConfiguration_py_uuid_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("uuid"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: uuid
void c_objc_pl_VZUSBDeviceConfiguration_py_uuid_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setUuid:"), val);
}

// Shim for Swift property getter: displays
void* c_objc_cs_VZMacGraphicsDeviceConfiguration_py_displays_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("displays"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: displays
void c_objc_cs_VZMacGraphicsDeviceConfiguration_py_displays_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setDisplays:"), val);
}

// Shim for Swift property getter: tag
void* c_objc_cs_VZVirtioFileSystemDevice_py_tag_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("tag"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: tag
void c_objc_cs_VZVirtioFileSystemDevice_py_tag_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setTag:"), val);
}

// Shim for Swift property getter: consoleDevices
void* c_objc_cs_VZVirtualMachine_py_consoleDevices_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("consoleDevices"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: consoleDevices
void c_objc_cs_VZVirtualMachine_py_consoleDevices_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setConsoleDevices:"), val);
}

// Shim for Swift property getter: networkDevices
void* c_objc_cs_VZVirtualMachineConfiguration_py_networkDevices_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("networkDevices"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: networkDevices
void c_objc_cs_VZVirtualMachineConfiguration_py_networkDevices_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setNetworkDevices:"), val);
}

// Shim for Swift property getter: ports
void* c_objc_cs_VZVirtioConsoleDevice_py_ports_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("ports"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: ports
void c_objc_cs_VZVirtioConsoleDevice_py_ports_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setPorts:"), val);
}

// Shim for Swift property getter: isForcedReadOnly
void* c_objc_cs_VZNetworkBlockDeviceStorageDeviceAttachment_py_forcedReadOnly_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("isForcedReadOnly"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: isForcedReadOnly
void c_objc_cs_VZNetworkBlockDeviceStorageDeviceAttachment_py_forcedReadOnly_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setIsForcedReadOnly:"), val);
}

// Shim for Swift property getter: restoreImageURL
void* c_objc_cs_VZMacOSInstaller_py_restoreImageURL_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("restoreImageURL"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: restoreImageURL
void c_objc_cs_VZMacOSInstaller_py_restoreImageURL_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setRestoreImageURL:"), val);
}

// Shim for Swift property getter: isReadOnly
void* c_objc_cs_VZSharedDirectory_py_readOnly_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("isReadOnly"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: isReadOnly
void c_objc_cs_VZSharedDirectory_py_readOnly_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setIsReadOnly:"), val);
}

// Shim for Swift property getter: cachingOptions
void* s_So28VZLinuxRosettaDirectoryShareC14VirtualizationE14cachingOptionsAbCE07CachingG0OSgvp_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("cachingOptions"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: cachingOptions
void s_So28VZLinuxRosettaDirectoryShareC14VirtualizationE14cachingOptionsAbCE07CachingG0OSgvp_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setCachingOptions:"), val);
}

// Shim for Swift property getter: dataRepresentation
void* c_objc_cs_VZMacMachineIdentifier_py_dataRepresentation_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("dataRepresentation"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: dataRepresentation
void c_objc_cs_VZMacMachineIdentifier_py_dataRepresentation_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setDataRepresentation:"), val);
}

// Shim for Swift property getter: url
void* c_objc_cs_VZMacAuxiliaryStorage_py_URL_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("url"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: url
void c_objc_cs_VZMacAuxiliaryStorage_py_URL_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setUrl:"), val);
}

// Shim for Swift method: attach(device:)
void* c_objc_cs_VZUSBController_im_attachDevice_completionHandler_(void* self, void* device) {
    id obj = (__bridge id)self;
    typedef void* (*MsgFn)(id, SEL, id, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj,
               sel_getUid("attach(device:):"),
               device);
    return (__bridge_retained void*)rv;
}

// Shim for Swift property getter: platform
void* c_objc_cs_VZVirtualMachineConfiguration_py_platform_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("platform"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: platform
void c_objc_cs_VZVirtualMachineConfiguration_py_platform_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setPlatform:"), val);
}

// Shim for Swift property getter: append
void* c_objc_cs_VZFileSerialPortAttachment_py_append_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("append"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: append
void c_objc_cs_VZFileSerialPortAttachment_py_append_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setAppend:"), val);
}

// Shim for Swift property getter: virtualMachine
void* c_objc_cs_VZMacOSInstaller_py_virtualMachine_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("virtualMachine"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: virtualMachine
void c_objc_cs_VZMacOSInstaller_py_virtualMachine_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setVirtualMachine:"), val);
}

// Shim for Swift property getter: errorCode
void* s_10Foundation21_BridgedStoredNSErrorPAAE9errorCodeSivp__SYNTHESIZED__s_SC11VZErrorCodeLeV_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("errorCode"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: errorCode
void s_10Foundation21_BridgedStoredNSErrorPAAE9errorCodeSivp__SYNTHESIZED__s_SC11VZErrorCodeLeV_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setErrorCode:"), val);
}

// Shim for Swift property getter: auxiliaryStorage
void* c_objc_cs_VZMacPlatformConfiguration_py_auxiliaryStorage_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("auxiliaryStorage"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: auxiliaryStorage
void c_objc_cs_VZMacPlatformConfiguration_py_auxiliaryStorage_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setAuxiliaryStorage:"), val);
}

// Shim for Swift property getter: heightInPixels
void* c_objc_cs_VZVirtioGraphicsScanoutConfiguration_py_heightInPixels_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("heightInPixels"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: heightInPixels
void c_objc_cs_VZVirtioGraphicsScanoutConfiguration_py_heightInPixels_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setHeightInPixels:"), val);
}

// Shim for Swift property getter: pixelsPerInch
void* c_objc_cs_VZMacGraphicsDisplayConfiguration_py_pixelsPerInch_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("pixelsPerInch"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: pixelsPerInch
void c_objc_cs_VZMacGraphicsDisplayConfiguration_py_pixelsPerInch_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setPixelsPerInch:"), val);
}

// Shim for Swift property getter: errorUserInfo
void* s_10Foundation21_BridgedStoredNSErrorPAAE13errorUserInfoSDySSypGvp__SYNTHESIZED__s_SC11VZErrorCodeLeV_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("errorUserInfo"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: errorUserInfo
void s_10Foundation21_BridgedStoredNSErrorPAAE13errorUserInfoSDySSypGvp__SYNTHESIZED__s_SC11VZErrorCodeLeV_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setErrorUserInfo:"), val);
}

// Shim for Swift property getter: machineIdentifier
void* c_objc_cs_VZMacPlatformConfiguration_py_machineIdentifier_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("machineIdentifier"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: machineIdentifier
void c_objc_cs_VZMacPlatformConfiguration_py_machineIdentifier_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setMachineIdentifier:"), val);
}

// Shim for Swift property getter: fileHandle
void* c_objc_cs_VZFileHandleNetworkDeviceAttachment_py_fileHandle_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("fileHandle"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: fileHandle
void c_objc_cs_VZFileHandleNetworkDeviceAttachment_py_fileHandle_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setFileHandle:"), val);
}

// Shim for Swift property getter: macAddress
void* c_objc_cs_VZNetworkDeviceConfiguration_py_MACAddress_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("macAddress"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: macAddress
void c_objc_cs_VZNetworkDeviceConfiguration_py_MACAddress_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setMacAddress:"), val);
}

// Shim for Swift method: stop()
void* c_objc_cs_VZVirtualMachine_im_stopWithCompletionHandler_(void* self) {
    id obj = (__bridge id)self;
    typedef void* (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("stop()"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift method: displayDidBeginReconfiguration(_:)
void* c_objc_pl_VZGraphicsDisplayObserver_im_displayDidBeginReconfiguration_(void* self, void* display) {
    id obj = (__bridge id)self;
    typedef void* (*MsgFn)(id, SEL, id, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj,
               sel_getUid("displayDidBeginReconfiguration(_:):"),
               display);
    return (__bridge_retained void*)rv;
}

// Shim for Swift property getter: delegate
void* c_objc_cs_VZVirtioSocketListener_py_delegate_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("delegate"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: delegate
void c_objc_cs_VZVirtioSocketListener_py_delegate_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setDelegate:"), val);
}

// Shim for Swift property getter: maximumTransmissionUnit
void* c_objc_cs_VZFileHandleNetworkDeviceAttachment_py_maximumTransmissionUnit_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("maximumTransmissionUnit"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: maximumTransmissionUnit
void c_objc_cs_VZFileHandleNetworkDeviceAttachment_py_maximumTransmissionUnit_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setMaximumTransmissionUnit:"), val);
}

// Shim for Swift method: displayDidEndReconfiguration(_:)
void* c_objc_pl_VZGraphicsDisplayObserver_im_displayDidEndReconfiguration_(void* self, void* display) {
    id obj = (__bridge id)self;
    typedef void* (*MsgFn)(id, SEL, id, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj,
               sel_getUid("displayDidEndReconfiguration(_:):"),
               display);
    return (__bridge_retained void*)rv;
}

// Shim for Swift method: setCachingOptions(_:)
void* s_So28VZLinuxRosettaDirectoryShareC14VirtualizationE17setCachingOptionsyyAbCE0gH0OSgKF(void* self, void* cachingOptions) {
    id obj = (__bridge id)self;
    typedef void* (*MsgFn)(id, SEL, id, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj,
               sel_getUid("setCachingOptions(_:):"),
               cachingOptions);
    return (__bridge_retained void*)rv;
}

// Shim for Swift property getter: url
void* c_objc_cs_VZFileSerialPortAttachment_py_URL_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("url"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: url
void c_objc_cs_VZFileSerialPortAttachment_py_URL_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setUrl:"), val);
}

// Shim for Swift property getter: heightInPixels
void* c_objc_cs_VZMacGraphicsDisplayConfiguration_py_heightInPixels_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("heightInPixels"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: heightInPixels
void c_objc_cs_VZMacGraphicsDisplayConfiguration_py_heightInPixels_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setHeightInPixels:"), val);
}

// Shim for Swift property getter: machineIdentifier
void* c_objc_cs_VZGenericPlatformConfiguration_py_machineIdentifier_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("machineIdentifier"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: machineIdentifier
void c_objc_cs_VZGenericPlatformConfiguration_py_machineIdentifier_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setMachineIdentifier:"), val);
}

// Shim for Swift property getter: attachment
void* c_objc_cs_VZConsolePortConfiguration_py_attachment_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("attachment"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: attachment
void c_objc_cs_VZConsolePortConfiguration_py_attachment_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setAttachment:"), val);
}

// Shim for Swift property getter: synchronizationMode
void* c_objc_cs_VZDiskImageStorageDeviceAttachment_py_synchronizationMode_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("synchronizationMode"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: synchronizationMode
void c_objc_cs_VZDiskImageStorageDeviceAttachment_py_synchronizationMode_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setSynchronizationMode:"), val);
}

// Shim for Swift method: removeObserver(_:)
void* c_objc_cs_VZGraphicsDisplay_im_removeObserver_(void* self, void* observer) {
    id obj = (__bridge id)self;
    typedef void* (*MsgFn)(id, SEL, id, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj,
               sel_getUid("removeObserver(_:):"),
               observer);
    return (__bridge_retained void*)rv;
}

// Shim for Swift property getter: hardwareModel
void* c_objc_cs_VZMacPlatformConfiguration_py_hardwareModel_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("hardwareModel"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: hardwareModel
void c_objc_cs_VZMacPlatformConfiguration_py_hardwareModel_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setHardwareModel:"), val);
}

// Shim for Swift property getter: directory
void* c_objc_cs_VZSingleDirectoryShare_py_directory_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("directory"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: directory
void c_objc_cs_VZSingleDirectoryShare_py_directory_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setDirectory:"), val);
}

// Shim for Swift property getter: audioDevices
void* c_objc_cs_VZVirtualMachineConfiguration_py_audioDevices_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("audioDevices"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: audioDevices
void c_objc_cs_VZVirtualMachineConfiguration_py_audioDevices_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setAudioDevices:"), val);
}

// Shim for Swift property getter: usbControllers
void* c_objc_cs_VZVirtualMachineConfiguration_py_usbControllers_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("usbControllers"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: usbControllers
void c_objc_cs_VZVirtualMachineConfiguration_py_usbControllers_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setUsbControllers:"), val);
}

// Shim for Swift property getter: variableStore
void* c_objc_cs_VZEFIBootLoader_py_variableStore_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("variableStore"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: variableStore
void c_objc_cs_VZEFIBootLoader_py_variableStore_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setVariableStore:"), val);
}

// Shim for Swift property getter: maximumPortCount
void* c_objc_cs_VZVirtioConsolePortArray_py_maximumPortCount_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("maximumPortCount"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: maximumPortCount
void c_objc_cs_VZVirtioConsolePortArray_py_maximumPortCount_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setMaximumPortCount:"), val);
}

// Shim for Swift property getter: attachment
void* c_objc_cs_VZNetworkDeviceConfiguration_py_attachment_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("attachment"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: attachment
void c_objc_cs_VZNetworkDeviceConfiguration_py_attachment_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setAttachment:"), val);
}

// Shim for Swift property getter: destinationPort
void* c_objc_cs_VZVirtioSocketConnection_py_destinationPort_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("destinationPort"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: destinationPort
void c_objc_cs_VZVirtioSocketConnection_py_destinationPort_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setDestinationPort:"), val);
}

// Shim for Swift method: removeSocketListener(forPort:)
void* c_objc_cs_VZVirtioSocketDevice_im_removeSocketListenerForPort_(void* self, void* port) {
    id obj = (__bridge id)self;
    typedef void* (*MsgFn)(id, SEL, id, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj,
               sel_getUid("removeSocketListener(forPort:):"),
               port);
    return (__bridge_retained void*)rv;
}

// Shim for Swift property getter: progress
void* c_objc_cs_VZMacOSInstaller_py_progress_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("progress"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: progress
void c_objc_cs_VZMacOSInstaller_py_progress_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setProgress:"), val);
}

// Shim for Swift method: start(options:)
void* c_objc_cs_VZVirtualMachine_im_startWithOptions_completionHandler_(void* self, void* options) {
    id obj = (__bridge id)self;
    typedef void* (*MsgFn)(id, SEL, id, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj,
               sel_getUid("start(options:):"),
               options);
    return (__bridge_retained void*)rv;
}

// Shim for Swift method: setSocketListener(_:forPort:)
void* c_objc_cs_VZVirtioSocketDevice_im_setSocketListener_forPort_(void* self, void* listener, void* port) {
    id obj = (__bridge id)self;
    typedef void* (*MsgFn)(id, SEL, id, id, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj,
               sel_getUid("setSocketListener(_:forPort:):"),
               listener, port);
    return (__bridge_retained void*)rv;
}

// Shim for Swift method: pause()
void* c_objc_cs_VZVirtualMachine_im_pauseWithCompletionHandler_(void* self) {
    id obj = (__bridge id)self;
    typedef void* (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("pause()"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property getter: sourcePort
void* c_objc_cs_VZVirtioSocketConnection_py_sourcePort_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("sourcePort"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: sourcePort
void c_objc_cs_VZVirtioSocketConnection_py_sourcePort_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setSourcePort:"), val);
}

// Shim for Swift method: attachmentWasConnected(_:)
void* c_objc_pl_VZNetworkBlockDeviceStorageDeviceAttachmentDelegate_im_attachmentWasConnected_(void* self, void* attachment) {
    id obj = (__bridge id)self;
    typedef void* (*MsgFn)(id, SEL, id, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj,
               sel_getUid("attachmentWasConnected(_:):"),
               attachment);
    return (__bridge_retained void*)rv;
}

// Shim for Swift property getter: code
void* s_10Foundation21_BridgedStoredNSErrorPAAE4code4CodeQzvp__SYNTHESIZED__s_SC11VZErrorCodeLeV_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("code"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: code
void s_10Foundation21_BridgedStoredNSErrorPAAE4code4CodeQzvp__SYNTHESIZED__s_SC11VZErrorCodeLeV_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setCode:"), val);
}

// Shim for Swift method: reconfigure(configuration:)
void* c_objc_cs_VZGraphicsDisplay_im_reconfigureWithConfiguration_error_(void* self, void* configuration) {
    id obj = (__bridge id)self;
    typedef void* (*MsgFn)(id, SEL, id, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj,
               sel_getUid("reconfigure(configuration:):"),
               configuration);
    return (__bridge_retained void*)rv;
}

// Shim for Swift property getter: memoryBalloonDevices
void* c_objc_cs_VZVirtualMachine_py_memoryBalloonDevices_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("memoryBalloonDevices"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: memoryBalloonDevices
void c_objc_cs_VZVirtualMachine_py_memoryBalloonDevices_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setMemoryBalloonDevices:"), val);
}

// Shim for Swift method: requestStop()
void* c_objc_cs_VZVirtualMachine_im_requestStopWithError_(void* self) {
    id obj = (__bridge id)self;
    typedef void* (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("requestStop()"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property getter: streams
void* c_objc_cs_VZVirtioSoundDeviceConfiguration_py_streams_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("streams"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: streams
void c_objc_cs_VZVirtioSoundDeviceConfiguration_py_streams_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setStreams:"), val);
}

// Shim for Swift property getter: isSupported
void* c_objc_cs_VZMacHardwareModel_py_supported_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("isSupported"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: isSupported
void c_objc_cs_VZMacHardwareModel_py_supported_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setIsSupported:"), val);
}

// Shim for Swift property getter: bootLoader
void* c_objc_cs_VZVirtualMachineConfiguration_py_bootLoader_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("bootLoader"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: bootLoader
void c_objc_cs_VZVirtualMachineConfiguration_py_bootLoader_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setBootLoader:"), val);
}

// Shim for Swift property getter: networkDevices
void* c_objc_cs_VZVirtualMachine_py_networkDevices_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("networkDevices"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: networkDevices
void c_objc_cs_VZVirtualMachine_py_networkDevices_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setNetworkDevices:"), val);
}

// Shim for Swift method: attachment(_:didEncounterError:)
void* c_objc_pl_VZNetworkBlockDeviceStorageDeviceAttachmentDelegate_im_attachment_didEncounterError_(void* self, void* attachment, void* error) {
    id obj = (__bridge id)self;
    typedef void* (*MsgFn)(id, SEL, id, id, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj,
               sel_getUid("attachment(_:didEncounterError:):"),
               attachment, error);
    return (__bridge_retained void*)rv;
}

// Shim for Swift property getter: maximumPortCount
void* c_objc_cs_VZVirtioConsolePortConfigurationArray_py_maximumPortCount_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("maximumPortCount"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: maximumPortCount
void c_objc_cs_VZVirtioConsolePortConfigurationArray_py_maximumPortCount_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setMaximumPortCount:"), val);
}

// Shim for Swift method: addObserver(_:)
void* c_objc_cs_VZGraphicsDisplay_im_addObserver_(void* self, void* observer) {
    id obj = (__bridge id)self;
    typedef void* (*MsgFn)(id, SEL, id, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj,
               sel_getUid("addObserver(_:):"),
               observer);
    return (__bridge_retained void*)rv;
}

// Shim for Swift method: close()
void* c_objc_cs_VZVirtioSocketConnection_im_close(void* self) {
    id obj = (__bridge id)self;
    typedef void* (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("close()"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property getter: startUpFromMacOSRecovery
void* c_objc_cs_VZMacOSVirtualMachineStartOptions_py_startUpFromMacOSRecovery_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("startUpFromMacOSRecovery"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: startUpFromMacOSRecovery
void c_objc_cs_VZMacOSVirtualMachineStartOptions_py_startUpFromMacOSRecovery_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setStartUpFromMacOSRecovery:"), val);
}

// Shim for Swift property getter: graphicsDevices
void* c_objc_cs_VZVirtualMachineConfiguration_py_graphicsDevices_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("graphicsDevices"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: graphicsDevices
void c_objc_cs_VZVirtualMachineConfiguration_py_graphicsDevices_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setGraphicsDevices:"), val);
}

// Shim for Swift property getter: dataRepresentation
void* c_objc_cs_VZMacHardwareModel_py_dataRepresentation_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("dataRepresentation"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: dataRepresentation
void c_objc_cs_VZMacHardwareModel_py_dataRepresentation_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setDataRepresentation:"), val);
}

// Shim for Swift property getter: fileDescriptor
void* c_objc_cs_VZVirtioSocketConnection_py_fileDescriptor_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("fileDescriptor"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: fileDescriptor
void c_objc_cs_VZVirtioSocketConnection_py_fileDescriptor_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setFileDescriptor:"), val);
}

// Shim for Swift property getter: interface
void* c_objc_cs_VZBridgedNetworkDeviceAttachment_py_interface_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("interface"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: interface
void c_objc_cs_VZBridgedNetworkDeviceAttachment_py_interface_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setInterface:"), val);
}

// Shim for Swift property getter: sharesClipboard
void* c_objc_cs_VZSpiceAgentPortAttachment_py_sharesClipboard_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("sharesClipboard"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: sharesClipboard
void c_objc_cs_VZSpiceAgentPortAttachment_py_sharesClipboard_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setSharesClipboard:"), val);
}

// Shim for Swift property getter: socketDevices
void* c_objc_cs_VZVirtualMachine_py_socketDevices_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("socketDevices"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: socketDevices
void c_objc_cs_VZVirtualMachine_py_socketDevices_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setSocketDevices:"), val);
}

// Shim for Swift property getter: name
void* c_objc_cs_VZVirtioConsolePort_py_name_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("name"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: name
void c_objc_cs_VZVirtioConsolePort_py_name_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setName:"), val);
}

// Shim for Swift method: start()
void* c_objc_cs_VZVirtualMachine_im_startWithCompletionHandler_(void* self) {
    id obj = (__bridge id)self;
    typedef void* (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("start()"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property getter: userInfo
void* s_10Foundation21_BridgedStoredNSErrorPAAE8userInfoSDySSypGvp__SYNTHESIZED__s_SC11VZErrorCodeLeV_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("userInfo"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: userInfo
void s_10Foundation21_BridgedStoredNSErrorPAAE8userInfoSDySSypGvp__SYNTHESIZED__s_SC11VZErrorCodeLeV_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setUserInfo:"), val);
}

// Shim for Swift property getter: url
void* c_objc_cs_VZSharedDirectory_py_URL_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("url"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: url
void c_objc_cs_VZSharedDirectory_py_URL_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setUrl:"), val);
}

// Shim for Swift property getter: name
void* c_objc_cs_VZVirtioConsolePortConfiguration_py_name_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("name"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: name
void c_objc_cs_VZVirtioConsolePortConfiguration_py_name_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setName:"), val);
}

// Shim for Swift method: listener(_:shouldAcceptNewConnection:from:)
void* c_objc_pl_VZVirtioSocketListenerDelegate_im_listener_shouldAcceptNewConnection_fromSocketDevice_(void* self, void* listener, void* connection, void* socketDevice) {
    id obj = (__bridge id)self;
    typedef void* (*MsgFn)(id, SEL, id, id, id, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj,
               sel_getUid("listener(_:shouldAcceptNewConnection:from:):"),
               listener, connection, socketDevice);
    return (__bridge_retained void*)rv;
}

// Shim for Swift property getter: memorySize
void* c_objc_cs_VZVirtualMachineConfiguration_py_memorySize_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("memorySize"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: memorySize
void c_objc_cs_VZVirtualMachineConfiguration_py_memorySize_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setMemorySize:"), val);
}

// Shim for Swift property getter: mostFeaturefulSupportedConfiguration
void* c_objc_cs_VZMacOSRestoreImage_py_mostFeaturefulSupportedConfiguration_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("mostFeaturefulSupportedConfiguration"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: mostFeaturefulSupportedConfiguration
void c_objc_cs_VZMacOSRestoreImage_py_mostFeaturefulSupportedConfiguration_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setMostFeaturefulSupportedConfiguration:"), val);
}

// Shim for Swift property getter: sizeInPixels
void* c_objc_cs_VZGraphicsDisplay_py_sizeInPixels_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("sizeInPixels"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: sizeInPixels
void c_objc_cs_VZGraphicsDisplay_py_sizeInPixels_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setSizeInPixels:"), val);
}

// Shim for Swift property getter: attachment
void* c_objc_cs_VZNetworkDevice_py_attachment_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("attachment"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: attachment
void c_objc_cs_VZNetworkDevice_py_attachment_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setAttachment:"), val);
}

// Shim for Swift property getter: widthInPixels
void* c_objc_cs_VZMacGraphicsDisplayConfiguration_py_widthInPixels_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("widthInPixels"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: widthInPixels
void c_objc_cs_VZMacGraphicsDisplayConfiguration_py_widthInPixels_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setWidthInPixels:"), val);
}

// Shim for Swift method: install()
void* c_objc_cs_VZMacOSInstaller_im_installWithCompletionHandler_(void* self) {
    id obj = (__bridge id)self;
    typedef void* (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("install()"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property getter: pointingDevices
void* c_objc_cs_VZVirtualMachineConfiguration_py_pointingDevices_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("pointingDevices"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: pointingDevices
void c_objc_cs_VZVirtualMachineConfiguration_py_pointingDevices_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setPointingDevices:"), val);
}

// Shim for Swift property getter: usbController
void* c_objc_pl_VZUSBDevice_py_usbController_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("usbController"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: usbController
void c_objc_pl_VZUSBDevice_py_usbController_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setUsbController:"), val);
}

// Shim for Swift method: reconfigure(sizeInPixels:)
void* c_objc_cs_VZGraphicsDisplay_im_reconfigureWithSizeInPixels_error_(void* self, void* sizeInPixels) {
    id obj = (__bridge id)self;
    typedef void* (*MsgFn)(id, SEL, id, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj,
               sel_getUid("reconfigure(sizeInPixels:):"),
               sizeInPixels);
    return (__bridge_retained void*)rv;
}

// Shim for Swift property getter: usbControllers
void* c_objc_cs_VZVirtualMachine_py_usbControllers_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("usbControllers"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: usbControllers
void c_objc_cs_VZVirtualMachine_py_usbControllers_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setUsbControllers:"), val);
}

// Shim for Swift property getter: isConsole
void* c_objc_cs_VZVirtioConsolePortConfiguration_py_isConsole_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("isConsole"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: isConsole
void c_objc_cs_VZVirtioConsolePortConfiguration_py_isConsole_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setIsConsole:"), val);
}

// Shim for Swift property getter: cpuCount
void* c_objc_cs_VZVirtualMachineConfiguration_py_CPUCount_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("cpuCount"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: cpuCount
void c_objc_cs_VZVirtualMachineConfiguration_py_CPUCount_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setCpuCount:"), val);
}

// Shim for Swift property getter: attachment
void* c_objc_cs_VZVirtioConsolePort_py_attachment_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("attachment"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: attachment
void c_objc_cs_VZVirtioConsolePort_py_attachment_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setAttachment:"), val);
}

// Shim for Swift property getter: url
void* c_objc_cs_VZMacOSRestoreImage_py_URL_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("url"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: url
void c_objc_cs_VZMacOSRestoreImage_py_URL_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setUrl:"), val);
}

// Shim for Swift method: virtualMachine(_:networkDevice:attachmentWasDisconnectedWithError:)
void* c_objc_pl_VZVirtualMachineDelegate_im_virtualMachine_networkDevice_attachmentWasDisconnectedWithError_(void* self, void* virtualMachine, void* networkDevice, void* error) {
    id obj = (__bridge id)self;
    typedef void* (*MsgFn)(id, SEL, id, id, id, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj,
               sel_getUid("virtualMachine(_:networkDevice:attachmentWasDisconnectedWithError:):"),
               virtualMachine, networkDevice, error);
    return (__bridge_retained void*)rv;
}

// Shim for Swift property getter: operatingSystemVersion
void* c_objc_cs_VZMacOSRestoreImage_py_operatingSystemVersion_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("operatingSystemVersion"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: operatingSystemVersion
void c_objc_cs_VZMacOSRestoreImage_py_operatingSystemVersion_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setOperatingSystemVersion:"), val);
}

// Shim for Swift property getter: entropyDevices
void* c_objc_cs_VZVirtualMachineConfiguration_py_entropyDevices_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("entropyDevices"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: entropyDevices
void c_objc_cs_VZVirtualMachineConfiguration_py_entropyDevices_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setEntropyDevices:"), val);
}

// Shim for Swift method: restoreMachineStateFrom(url:)
void* c_objc_cs_VZVirtualMachine_im_restoreMachineStateFromURL_completionHandler_(void* self, void* saveFileURL) {
    id obj = (__bridge id)self;
    typedef void* (*MsgFn)(id, SEL, id, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj,
               sel_getUid("restoreMachineStateFrom(url:):"),
               saveFileURL);
    return (__bridge_retained void*)rv;
}

// Shim for Swift property getter: identifier
void* c_objc_cs_VZBridgedNetworkInterface_py_identifier_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("identifier"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: identifier
void c_objc_cs_VZBridgedNetworkInterface_py_identifier_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setIdentifier:"), val);
}

// Shim for Swift property getter: tag
void* c_objc_cs_VZVirtioFileSystemDeviceConfiguration_py_tag_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("tag"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: tag
void c_objc_cs_VZVirtioFileSystemDeviceConfiguration_py_tag_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setTag:"), val);
}

// Shim for Swift property getter: isUnicastAddress
void* c_objc_cs_VZMACAddress_py_isUnicastAddress_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("isUnicastAddress"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: isUnicastAddress
void c_objc_cs_VZMACAddress_py_isUnicastAddress_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setIsUnicastAddress:"), val);
}

// Shim for Swift property getter: url
void* c_objc_cs_VZEFIVariableStore_py_URL_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("url"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: url
void c_objc_cs_VZEFIVariableStore_py_URL_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setUrl:"), val);
}

// Shim for Swift property getter: isUniversallyAdministeredAddress
void* c_objc_cs_VZMACAddress_py_isUniversallyAdministeredAddress_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("isUniversallyAdministeredAddress"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: isUniversallyAdministeredAddress
void c_objc_cs_VZMACAddress_py_isUniversallyAdministeredAddress_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setIsUniversallyAdministeredAddress:"), val);
}

// Shim for Swift property getter: capturesSystemKeys
void* c_objc_cs_VZVirtualMachineView_py_capturesSystemKeys_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("capturesSystemKeys"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: capturesSystemKeys
void c_objc_cs_VZVirtualMachineView_py_capturesSystemKeys_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setCapturesSystemKeys:"), val);
}

// Shim for Swift property getter: isLocallyAdministeredAddress
void* c_objc_cs_VZMACAddress_py_isLocallyAdministeredAddress_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("isLocallyAdministeredAddress"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: isLocallyAdministeredAddress
void c_objc_cs_VZMACAddress_py_isLocallyAdministeredAddress_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setIsLocallyAdministeredAddress:"), val);
}

// Shim for Swift method: virtualMachine(_:didStopWithError:)
void* c_objc_pl_VZVirtualMachineDelegate_im_virtualMachine_didStopWithError_(void* self, void* virtualMachine, void* error) {
    id obj = (__bridge id)self;
    typedef void* (*MsgFn)(id, SEL, id, id, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj,
               sel_getUid("virtualMachine(_:didStopWithError:):"),
               virtualMachine, error);
    return (__bridge_retained void*)rv;
}

// Shim for Swift property getter: synchronizationMode
void* c_objc_cs_VZNetworkBlockDeviceStorageDeviceAttachment_py_synchronizationMode_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("synchronizationMode"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: synchronizationMode
void c_objc_cs_VZNetworkBlockDeviceStorageDeviceAttachment_py_synchronizationMode_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setSynchronizationMode:"), val);
}

// Shim for Swift property getter: source
void* c_objc_cs_VZVirtioSoundDeviceInputStreamConfiguration_py_source_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("source"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: source
void c_objc_cs_VZVirtioSoundDeviceInputStreamConfiguration_py_source_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setSource:"), val);
}

// Shim for Swift property getter: ports
void* c_objc_cs_VZVirtioConsoleDeviceConfiguration_py_ports_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("ports"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: ports
void c_objc_cs_VZVirtioConsoleDeviceConfiguration_py_ports_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setPorts:"), val);
}

// Shim for Swift property getter: delegate
void* c_objc_cs_VZVirtualMachine_py_delegate_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("delegate"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: delegate
void c_objc_cs_VZVirtualMachine_py_delegate_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setDelegate:"), val);
}

// Shim for Swift property getter: delegate
void* c_objc_cs_VZNetworkBlockDeviceStorageDeviceAttachment_py_delegate_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("delegate"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: delegate
void c_objc_cs_VZNetworkBlockDeviceStorageDeviceAttachment_py_delegate_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setDelegate:"), val);
}

// Shim for Swift method: consoleDevice(_:didOpen:)
void* c_objc_pl_VZVirtioConsoleDeviceDelegate_im_consoleDevice_didOpenPort_(void* self, void* consoleDevice, void* consolePort) {
    id obj = (__bridge id)self;
    typedef void* (*MsgFn)(id, SEL, id, id, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj,
               sel_getUid("consoleDevice(_:didOpen:):"),
               consoleDevice, consolePort);
    return (__bridge_retained void*)rv;
}

// Shim for Swift property getter: dataRepresentation
void* c_objc_cs_VZGenericMachineIdentifier_py_dataRepresentation_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("dataRepresentation"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: dataRepresentation
void c_objc_cs_VZGenericMachineIdentifier_py_dataRepresentation_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setDataRepresentation:"), val);
}

// Shim for Swift property getter: isBroadcastAddress
void* c_objc_cs_VZMACAddress_py_isBroadcastAddress_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("isBroadcastAddress"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: isBroadcastAddress
void c_objc_cs_VZMACAddress_py_isBroadcastAddress_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setIsBroadcastAddress:"), val);
}

// Shim for Swift property getter: keyboards
void* c_objc_cs_VZVirtualMachineConfiguration_py_keyboards_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("keyboards"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: keyboards
void c_objc_cs_VZVirtualMachineConfiguration_py_keyboards_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setKeyboards:"), val);
}

// Shim for Swift property getter: sink
void* c_objc_cs_VZVirtioSoundDeviceOutputStreamConfiguration_py_sink_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("sink"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: sink
void c_objc_cs_VZVirtioSoundDeviceOutputStreamConfiguration_py_sink_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setSink:"), val);
}

// Shim for Swift property getter: pixelsPerInch
void* c_objc_cs_VZMacGraphicsDisplay_py_pixelsPerInch_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("pixelsPerInch"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: pixelsPerInch
void c_objc_cs_VZMacGraphicsDisplay_py_pixelsPerInch_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setPixelsPerInch:"), val);
}

// Shim for Swift property getter: share
void* c_objc_cs_VZVirtioFileSystemDeviceConfiguration_py_share_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("share"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: share
void c_objc_cs_VZVirtioFileSystemDeviceConfiguration_py_share_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setShare:"), val);
}

// Shim for Swift property getter: localizedDisplayName
void* c_objc_cs_VZBridgedNetworkInterface_py_localizedDisplayName_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("localizedDisplayName"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: localizedDisplayName
void c_objc_cs_VZBridgedNetworkInterface_py_localizedDisplayName_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setLocalizedDisplayName:"), val);
}

// Shim for Swift property getter: isMulticastAddress
void* c_objc_cs_VZMACAddress_py_isMulticastAddress_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("isMulticastAddress"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: isMulticastAddress
void c_objc_cs_VZMACAddress_py_isMulticastAddress_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setIsMulticastAddress:"), val);
}

// Shim for Swift property getter: isSupported
void* c_objc_cs_VZMacOSRestoreImage_py_supported_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("isSupported"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: isSupported
void c_objc_cs_VZMacOSRestoreImage_py_supported_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setIsSupported:"), val);
}

// Shim for Swift property getter: directories
void* c_objc_cs_VZMultipleDirectoryShare_py_directories_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("directories"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: directories
void c_objc_cs_VZMultipleDirectoryShare_py_directories_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setDirectories:"), val);
}

// Shim for Swift property getter: displays
void* c_objc_cs_VZGraphicsDevice_py_displays_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("displays"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: displays
void c_objc_cs_VZGraphicsDevice_py_displays_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setDisplays:"), val);
}

// Shim for Swift property getter: virtualMachine
void* c_objc_cs_VZVirtualMachineView_py_virtualMachine_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("virtualMachine"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: virtualMachine
void c_objc_cs_VZVirtualMachineView_py_virtualMachine_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setVirtualMachine:"), val);
}

// Shim for Swift property getter: buildVersion
void* c_objc_cs_VZMacOSRestoreImage_py_buildVersion_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("buildVersion"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: buildVersion
void c_objc_cs_VZMacOSRestoreImage_py_buildVersion_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setBuildVersion:"), val);
}

// Shim for Swift property getter: state
void* c_objc_cs_VZVirtualMachine_py_state_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("state"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: state
void c_objc_cs_VZVirtualMachine_py_state_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setState:"), val);
}

// Shim for Swift property getter: kernelURL
void* c_objc_cs_VZLinuxBootLoader_py_kernelURL_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("kernelURL"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: kernelURL
void c_objc_cs_VZLinuxBootLoader_py_kernelURL_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setKernelURL:"), val);
}

// Shim for Swift method: guestDidStop(_:)
void* c_objc_pl_VZVirtualMachineDelegate_im_guestDidStopVirtualMachine_(void* self, void* virtualMachine) {
    id obj = (__bridge id)self;
    typedef void* (*MsgFn)(id, SEL, id, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj,
               sel_getUid("guestDidStop(_:):"),
               virtualMachine);
    return (__bridge_retained void*)rv;
}

// Shim for Swift property getter: string
void* c_objc_cs_VZMACAddress_py_string_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("string"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: string
void c_objc_cs_VZMACAddress_py_string_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setString:"), val);
}

// Shim for Swift property getter: fileHandleForReading
void* c_objc_cs_VZFileHandleSerialPortAttachment_py_fileHandleForReading_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("fileHandleForReading"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: fileHandleForReading
void c_objc_cs_VZFileHandleSerialPortAttachment_py_fileHandleForReading_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setFileHandleForReading:"), val);
}

// Shim for Swift property getter: targetVirtualMachineMemorySize
void* c_objc_cs_VZVirtioTraditionalMemoryBalloonDevice_py_targetVirtualMachineMemorySize_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("targetVirtualMachineMemorySize"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: targetVirtualMachineMemorySize
void c_objc_cs_VZVirtioTraditionalMemoryBalloonDevice_py_targetVirtualMachineMemorySize_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setTargetVirtualMachineMemorySize:"), val);
}

// Shim for Swift property getter: usbDevices
void* c_objc_cs_VZUSBControllerConfiguration_py_usbDevices_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("usbDevices"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: usbDevices
void c_objc_cs_VZUSBControllerConfiguration_py_usbDevices_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setUsbDevices:"), val);
}

// Shim for Swift property getter: socketDevices
void* c_objc_cs_VZVirtualMachineConfiguration_py_socketDevices_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("socketDevices"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: socketDevices
void c_objc_cs_VZVirtualMachineConfiguration_py_socketDevices_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setSocketDevices:"), val);
}

// Shim for Swift property getter: blockDeviceIdentifier
void* c_objc_cs_VZVirtioBlockDeviceConfiguration_py_blockDeviceIdentifier_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("blockDeviceIdentifier"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: blockDeviceIdentifier
void c_objc_cs_VZVirtioBlockDeviceConfiguration_py_blockDeviceIdentifier_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setBlockDeviceIdentifier:"), val);
}

// Shim for Swift property getter: canStop
void* c_objc_cs_VZVirtualMachine_py_canStop_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("canStop"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: canStop
void c_objc_cs_VZVirtualMachine_py_canStop_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setCanStop:"), val);
}

// Shim for Swift method: consoleDevice(_:didClose:)
void* c_objc_pl_VZVirtioConsoleDeviceDelegate_im_consoleDevice_didClosePort_(void* self, void* consoleDevice, void* consolePort) {
    id obj = (__bridge id)self;
    typedef void* (*MsgFn)(id, SEL, id, id, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj,
               sel_getUid("consoleDevice(_:didClose:):"),
               consoleDevice, consolePort);
    return (__bridge_retained void*)rv;
}

// Shim for Swift property getter: graphicsDevices
void* c_objc_cs_VZVirtualMachine_py_graphicsDevices_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("graphicsDevices"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: graphicsDevices
void c_objc_cs_VZVirtualMachine_py_graphicsDevices_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setGraphicsDevices:"), val);
}

// Shim for Swift method: saveMachineStateTo(url:)
void* c_objc_cs_VZVirtualMachine_im_saveMachineStateToURL_completionHandler_(void* self, void* saveFileURL) {
    id obj = (__bridge id)self;
    typedef void* (*MsgFn)(id, SEL, id, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj,
               sel_getUid("saveMachineStateTo(url:):"),
               saveFileURL);
    return (__bridge_retained void*)rv;
}

// Shim for Swift property getter: commandLine
void* c_objc_cs_VZLinuxBootLoader_py_commandLine_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("commandLine"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: commandLine
void c_objc_cs_VZLinuxBootLoader_py_commandLine_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setCommandLine:"), val);
}

// Shim for Swift property getter: canStart
void* c_objc_cs_VZVirtualMachine_py_canStart_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("canStart"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: canStart
void c_objc_cs_VZVirtualMachine_py_canStart_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setCanStart:"), val);
}

// Shim for Swift property getter: ethernetAddress
void* c_objc_cs_VZMACAddress_py_ethernetAddress_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("ethernetAddress"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: ethernetAddress
void c_objc_cs_VZMACAddress_py_ethernetAddress_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setEthernetAddress:"), val);
}

// Shim for Swift property getter: automaticallyReconfiguresDisplay
void* c_objc_cs_VZVirtualMachineView_py_automaticallyReconfiguresDisplay_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("automaticallyReconfiguresDisplay"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: automaticallyReconfiguresDisplay
void c_objc_cs_VZVirtualMachineView_py_automaticallyReconfiguresDisplay_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setAutomaticallyReconfiguresDisplay:"), val);
}

// Shim for Swift property getter: storageDevices
void* c_objc_cs_VZVirtualMachineConfiguration_py_storageDevices_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("storageDevices"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: storageDevices
void c_objc_cs_VZVirtualMachineConfiguration_py_storageDevices_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setStorageDevices:"), val);
}

// Shim for Swift property getter: attachment
void* c_objc_cs_VZSerialPortConfiguration_py_attachment_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("attachment"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: attachment
void c_objc_cs_VZSerialPortConfiguration_py_attachment_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setAttachment:"), val);
}


#ifdef __cplusplus
}
#endif

