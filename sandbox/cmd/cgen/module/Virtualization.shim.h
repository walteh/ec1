#ifndef VIRTUALIZATION_SHIM_H
#define VIRTUALIZATION_SHIM_H

#include <stdint.h>

#ifdef __cplusplus
extern "C" {
#endif

// Swift type definitions
// VZNetworkDeviceConfiguration is a Swift class
typedef void* VZNetworkDeviceConfiguration;

// VZVirtioConsolePort is a Swift class
typedef void* VZVirtioConsolePort;

// VZVirtioSoundDeviceInputStreamConfiguration is a Swift class
typedef void* VZVirtioSoundDeviceInputStreamConfiguration;

// VZConsolePortConfiguration is a Swift class
typedef void* VZConsolePortConfiguration;

// VZBridgedNetworkInterface is a Swift class
typedef void* VZBridgedNetworkInterface;

// VZVirtioSoundDeviceConfiguration is a Swift class
typedef void* VZVirtioSoundDeviceConfiguration;

// VZSerialPortConfiguration is a Swift class
typedef void* VZSerialPortConfiguration;

// VZLinuxRosettaDirectoryShare.CachingOptions represents a Swift enum
typedef enum {
    VZLinuxRosettaDirectoryShare_CachingOptions_Unknown = 0,
} VZLinuxRosettaDirectoryShare_CachingOptions;

// VZVirtualMachine.State represents a Swift enum
typedef enum {
    VZVirtualMachine_State_Unknown = 0,
} VZVirtualMachine_State;

// VZUSBMassStorageDevice is a Swift class
typedef void* VZUSBMassStorageDevice;

// VZMacPlatformConfiguration is a Swift class
typedef void* VZMacPlatformConfiguration;

// VZDirectoryShare is a Swift class
typedef void* VZDirectoryShare;

// VZLinuxRosettaDirectoryShare is a Swift class
typedef void* VZLinuxRosettaDirectoryShare;

// VZVirtioSocketListenerDelegate is a Swift protocol
typedef void* VZVirtioSocketListenerDelegate;

// VZVirtioSoundDeviceOutputStreamConfiguration is a Swift class
typedef void* VZVirtioSoundDeviceOutputStreamConfiguration;

// VZSharedDirectory is a Swift class
typedef void* VZSharedDirectory;

// VZVirtioConsolePortConfiguration is a Swift class
typedef void* VZVirtioConsolePortConfiguration;

// VZUSBMassStorageDeviceConfiguration is a Swift class
typedef void* VZUSBMassStorageDeviceConfiguration;

// VZDiskImageSynchronizationMode represents a Swift enum
typedef enum {
    VZDiskImageSynchronizationMode_Unknown = 0,
} VZDiskImageSynchronizationMode;

// VZVirtualMachineDelegate is a Swift protocol
typedef void* VZVirtualMachineDelegate;

// VZConsoleDeviceConfiguration is a Swift class
typedef void* VZConsoleDeviceConfiguration;

// VZVirtioConsolePortArray is a Swift class
typedef void* VZVirtioConsolePortArray;

// VZDiskImageStorageDeviceAttachment is a Swift class
typedef void* VZDiskImageStorageDeviceAttachment;

// VZSingleDirectoryShare is a Swift class
typedef void* VZSingleDirectoryShare;

// VZVirtualMachineConfiguration is a Swift class
typedef void* VZVirtualMachineConfiguration;

// VZVirtioSoundDeviceStreamConfiguration is a Swift class
typedef void* VZVirtioSoundDeviceStreamConfiguration;

// VZUSBScreenCoordinatePointingDeviceConfiguration is a Swift class
typedef void* VZUSBScreenCoordinatePointingDeviceConfiguration;

// VZConsoleDevice is a Swift class
typedef void* VZConsoleDevice;

// VZUSBKeyboardConfiguration is a Swift class
typedef void* VZUSBKeyboardConfiguration;

// VZVirtioNetworkDeviceConfiguration is a Swift class
typedef void* VZVirtioNetworkDeviceConfiguration;

// VZVirtualMachine is a Swift class
typedef void* VZVirtualMachine;

// VZHostAudioInputStreamSource is a Swift class
typedef void* VZHostAudioInputStreamSource;

// VZAudioInputStreamSource is a Swift class
typedef void* VZAudioInputStreamSource;

// VZEntropyDeviceConfiguration is a Swift class
typedef void* VZEntropyDeviceConfiguration;

// VZMemoryBalloonDevice is a Swift class
typedef void* VZMemoryBalloonDevice;

// VZVirtioBlockDeviceConfiguration is a Swift class
typedef void* VZVirtioBlockDeviceConfiguration;

// VZVirtioConsolePortConfigurationArray is a Swift class
typedef void* VZVirtioConsolePortConfigurationArray;

// VZNetworkBlockDeviceStorageDeviceAttachment is a Swift class
typedef void* VZNetworkBlockDeviceStorageDeviceAttachment;

// VZVirtioGraphicsScanoutConfiguration is a Swift class
typedef void* VZVirtioGraphicsScanoutConfiguration;

// VZDiskSynchronizationMode represents a Swift enum
typedef enum {
    VZDiskSynchronizationMode_Unknown = 0,
} VZDiskSynchronizationMode;

// VZError represents a Swift struct
typedef struct {
    void* _internal;
} VZError;

// VZLinuxRosettaAvailability represents a Swift enum
typedef enum {
    VZLinuxRosettaAvailability_Unknown = 0,
} VZLinuxRosettaAvailability;

// VZMacKeyboardConfiguration is a Swift class
typedef void* VZMacKeyboardConfiguration;

// VZAudioOutputStreamSink is a Swift class
typedef void* VZAudioOutputStreamSink;

// VZVirtioConsoleDevice is a Swift class
typedef void* VZVirtioConsoleDevice;

// VZMemoryBalloonDeviceConfiguration is a Swift class
typedef void* VZMemoryBalloonDeviceConfiguration;

// VZPointingDeviceConfiguration is a Swift class
typedef void* VZPointingDeviceConfiguration;

// VZVirtioGraphicsScanout is a Swift class
typedef void* VZVirtioGraphicsScanout;

// VZNetworkBlockDeviceStorageDeviceAttachmentDelegate is a Swift protocol
typedef void* VZNetworkBlockDeviceStorageDeviceAttachmentDelegate;

// VZMacTrackpadConfiguration is a Swift class
typedef void* VZMacTrackpadConfiguration;

// VZMacHardwareModel is a Swift class
typedef void* VZMacHardwareModel;

// VZUSBDevice is a Swift protocol
typedef void* VZUSBDevice;

// VZVirtioSocketListener is a Swift class
typedef void* VZVirtioSocketListener;

// VZVirtioConsoleDeviceConfiguration is a Swift class
typedef void* VZVirtioConsoleDeviceConfiguration;

// VZVirtioFileSystemDevice is a Swift class
typedef void* VZVirtioFileSystemDevice;

// VZDirectorySharingDevice is a Swift class
typedef void* VZDirectorySharingDevice;

// VZBootLoader is a Swift class
typedef void* VZBootLoader;

// VZNetworkDevice is a Swift class
typedef void* VZNetworkDevice;

// VZNetworkDeviceAttachment is a Swift class
typedef void* VZNetworkDeviceAttachment;

// VZVirtioGraphicsDeviceConfiguration is a Swift class
typedef void* VZVirtioGraphicsDeviceConfiguration;

// VZXHCIController is a Swift class
typedef void* VZXHCIController;

// VZUSBDeviceConfiguration is a Swift protocol
typedef void* VZUSBDeviceConfiguration;

// VZMacGraphicsDisplayConfiguration is a Swift class
typedef void* VZMacGraphicsDisplayConfiguration;

// VZVirtioConsoleDeviceDelegate is a Swift protocol
typedef void* VZVirtioConsoleDeviceDelegate;

// VZVirtioEntropyDeviceConfiguration is a Swift class
typedef void* VZVirtioEntropyDeviceConfiguration;

// VZBridgedNetworkDeviceAttachment is a Swift class
typedef void* VZBridgedNetworkDeviceAttachment;

// VZXHCIControllerConfiguration is a Swift class
typedef void* VZXHCIControllerConfiguration;

// VZVirtioConsoleDeviceSerialPortConfiguration is a Swift class
typedef void* VZVirtioConsoleDeviceSerialPortConfiguration;

// VZFileSerialPortAttachment is a Swift class
typedef void* VZFileSerialPortAttachment;

// VZHostAudioOutputStreamSink is a Swift class
typedef void* VZHostAudioOutputStreamSink;

// VZMacGraphicsDisplay is a Swift class
typedef void* VZMacGraphicsDisplay;

// VZVirtioGraphicsDevice is a Swift class
typedef void* VZVirtioGraphicsDevice;

// VZUSBController is a Swift class
typedef void* VZUSBController;

// VZGraphicsDevice is a Swift class
typedef void* VZGraphicsDevice;

// VZMacMachineIdentifier is a Swift class
typedef void* VZMacMachineIdentifier;

// VZNVMExpressControllerDeviceConfiguration is a Swift class
typedef void* VZNVMExpressControllerDeviceConfiguration;

// VZVirtioSocketDeviceConfiguration is a Swift class
typedef void* VZVirtioSocketDeviceConfiguration;

// VZMacGraphicsDeviceConfiguration is a Swift class
typedef void* VZMacGraphicsDeviceConfiguration;

// VZVirtioFileSystemDeviceConfiguration is a Swift class
typedef void* VZVirtioFileSystemDeviceConfiguration;

// VZUSBControllerConfiguration is a Swift class
typedef void* VZUSBControllerConfiguration;

// VZStorageDeviceConfiguration is a Swift class
typedef void* VZStorageDeviceConfiguration;

// VZPlatformConfiguration is a Swift class
typedef void* VZPlatformConfiguration;

// VZNATNetworkDeviceAttachment is a Swift class
typedef void* VZNATNetworkDeviceAttachment;

// VZVirtioSocketDevice is a Swift class
typedef void* VZVirtioSocketDevice;

// VZLinuxBootLoader is a Swift class
typedef void* VZLinuxBootLoader;

// VZError.Code represents a Swift enum
typedef enum {
    VZError_Code_Unknown = 0,
} VZError_Code;

// VZFileHandleSerialPortAttachment is a Swift class
typedef void* VZFileHandleSerialPortAttachment;

// VZMacGraphicsDevice is a Swift class
typedef void* VZMacGraphicsDevice;

// VZSpiceAgentPortAttachment is a Swift class
typedef void* VZSpiceAgentPortAttachment;

// VZSerialPortAttachment is a Swift class
typedef void* VZSerialPortAttachment;

// VZGenericPlatformConfiguration is a Swift class
typedef void* VZGenericPlatformConfiguration;

// VZMacOSConfigurationRequirements is a Swift class
typedef void* VZMacOSConfigurationRequirements;

// VZMultipleDirectoryShare is a Swift class
typedef void* VZMultipleDirectoryShare;

// VZVirtioSocketConnection is a Swift class
typedef void* VZVirtioSocketConnection;

// VZKeyboardConfiguration is a Swift class
typedef void* VZKeyboardConfiguration;

// VZFileHandleNetworkDeviceAttachment is a Swift class
typedef void* VZFileHandleNetworkDeviceAttachment;

// VZStorageDevice is a Swift class
typedef void* VZStorageDevice;

// VZMacOSBootLoader is a Swift class
typedef void* VZMacOSBootLoader;

// VZDiskImageCachingMode represents a Swift enum
typedef enum {
    VZDiskImageCachingMode_Unknown = 0,
} VZDiskImageCachingMode;

// VZEFIVariableStore.InitializationOptions represents a Swift struct
typedef struct {
    void* _internal;
} VZEFIVariableStore_InitializationOptions;

// VZVirtioTraditionalMemoryBalloonDeviceConfiguration is a Swift class
typedef void* VZVirtioTraditionalMemoryBalloonDeviceConfiguration;

// VZGraphicsDisplayObserver is a Swift protocol
typedef void* VZGraphicsDisplayObserver;

// VZSocketDevice is a Swift class
typedef void* VZSocketDevice;

// VZMacAuxiliaryStorage.InitializationOptions represents a Swift struct
typedef struct {
    void* _internal;
} VZMacAuxiliaryStorage_InitializationOptions;

// VZMacOSRestoreImage is a Swift class
typedef void* VZMacOSRestoreImage;

// VZStorageDeviceAttachment is a Swift class
typedef void* VZStorageDeviceAttachment;

// VZDiskBlockDeviceStorageDeviceAttachment is a Swift class
typedef void* VZDiskBlockDeviceStorageDeviceAttachment;

// VZVirtioTraditionalMemoryBalloonDevice is a Swift class
typedef void* VZVirtioTraditionalMemoryBalloonDevice;

// VZVirtualMachineView is a Swift class
typedef void* VZVirtualMachineView;

// VZGraphicsDisplayConfiguration is a Swift class
typedef void* VZGraphicsDisplayConfiguration;

// VZSocketDeviceConfiguration is a Swift class
typedef void* VZSocketDeviceConfiguration;

// VZMacOSInstaller is a Swift class
typedef void* VZMacOSInstaller;

// VZDirectorySharingDeviceConfiguration is a Swift class
typedef void* VZDirectorySharingDeviceConfiguration;

// VZMACAddress is a Swift class
typedef void* VZMACAddress;

// VZEFIBootLoader is a Swift class
typedef void* VZEFIBootLoader;

// VZGraphicsDisplay is a Swift class
typedef void* VZGraphicsDisplay;

// VZAudioDeviceConfiguration is a Swift class
typedef void* VZAudioDeviceConfiguration;

// VZGenericMachineIdentifier is a Swift class
typedef void* VZGenericMachineIdentifier;

// VZEFIVariableStore is a Swift class
typedef void* VZEFIVariableStore;

// VZMacAuxiliaryStorage is a Swift class
typedef void* VZMacAuxiliaryStorage;

// VZVirtualMachineStartOptions is a Swift class
typedef void* VZVirtualMachineStartOptions;

// VZMacOSVirtualMachineStartOptions is a Swift class
typedef void* VZMacOSVirtualMachineStartOptions;

// VZGraphicsDeviceConfiguration is a Swift class
typedef void* VZGraphicsDeviceConfiguration;


// Swift property: streams
void* c_objc_cs_VZVirtioSoundDeviceConfiguration_py_streams_get(void* self);
void c_objc_cs_VZVirtioSoundDeviceConfiguration_py_streams_set(void* self, void* value);

// Swift property: attachment
void* c_objc_cs_VZConsolePortConfiguration_py_attachment_get(void* self);
void c_objc_cs_VZConsolePortConfiguration_py_attachment_set(void* self, void* value);

// Swift method: addObserver(_:)
void* c_objc_cs_VZGraphicsDisplay_im_addObserver_(void* self, void* observer);

// Swift property: tag
void* c_objc_cs_VZVirtioFileSystemDeviceConfiguration_py_tag_get(void* self);
void c_objc_cs_VZVirtioFileSystemDeviceConfiguration_py_tag_set(void* self, void* value);

// Swift property: isForcedReadOnly
void* c_objc_cs_VZNetworkBlockDeviceStorageDeviceAttachment_py_forcedReadOnly_get(void* self);
void c_objc_cs_VZNetworkBlockDeviceStorageDeviceAttachment_py_forcedReadOnly_set(void* self, void* value);

// Swift method: validate()
void* c_objc_cs_VZVirtualMachineConfiguration_im_validateWithError_(void* self);

// Swift property: widthInPixels
void* c_objc_cs_VZMacGraphicsDisplayConfiguration_py_widthInPixels_get(void* self);
void c_objc_cs_VZMacGraphicsDisplayConfiguration_py_widthInPixels_set(void* self, void* value);

// Swift method: restoreMachineStateFrom(url:)
void* c_objc_cs_VZVirtualMachine_im_restoreMachineStateFromURL_completionHandler_(void* self, void* saveFileURL);

// Swift property: ports
void* c_objc_cs_VZVirtioConsoleDeviceConfiguration_py_ports_get(void* self);
void c_objc_cs_VZVirtioConsoleDeviceConfiguration_py_ports_set(void* self, void* value);

// Swift method: detach(device:)
void* c_objc_cs_VZUSBController_im_detachDevice_completionHandler_(void* self, void* device);

// Swift method: setCachingOptions(_:)
void* s_So28VZLinuxRosettaDirectoryShareC14VirtualizationE17setCachingOptionsyyAbCE0gH0OSgKF(void* self, void* cachingOptions);

// Swift property: entropyDevices
void* c_objc_cs_VZVirtualMachineConfiguration_py_entropyDevices_get(void* self);
void c_objc_cs_VZVirtualMachineConfiguration_py_entropyDevices_set(void* self, void* value);

// Swift method: validateSaveRestoreSupport()
void* c_objc_cs_VZVirtualMachineConfiguration_im_validateSaveRestoreSupportWithError_(void* self);

// Swift property: timeout
void* c_objc_cs_VZNetworkBlockDeviceStorageDeviceAttachment_py_timeout_get(void* self);
void c_objc_cs_VZNetworkBlockDeviceStorageDeviceAttachment_py_timeout_set(void* self, void* value);

// Swift property: hardwareModel
void* c_objc_cs_VZMacPlatformConfiguration_py_hardwareModel_get(void* self);
void c_objc_cs_VZMacPlatformConfiguration_py_hardwareModel_set(void* self, void* value);

// Swift property: usbDevices
void* c_objc_cs_VZUSBController_py_usbDevices_get(void* self);
void c_objc_cs_VZUSBController_py_usbDevices_set(void* self, void* value);

// Swift property: share
void* c_objc_cs_VZVirtioFileSystemDeviceConfiguration_py_share_get(void* self);
void c_objc_cs_VZVirtioFileSystemDeviceConfiguration_py_share_set(void* self, void* value);

// Swift property: cachingOptions
void* s_So28VZLinuxRosettaDirectoryShareC14VirtualizationE14cachingOptionsAbCE07CachingG0OSgvp_get(void* self);
void s_So28VZLinuxRosettaDirectoryShareC14VirtualizationE14cachingOptionsAbCE07CachingG0OSgvp_set(void* self, void* value);

// Swift property: ports
void* c_objc_cs_VZVirtioConsoleDevice_py_ports_get(void* self);
void c_objc_cs_VZVirtioConsoleDevice_py_ports_set(void* self, void* value);

// Swift property: code
void* s_10Foundation21_BridgedStoredNSErrorPAAE4code4CodeQzvp__SYNTHESIZED__s_SC11VZErrorCodeLeV_get(void* self);
void s_10Foundation21_BridgedStoredNSErrorPAAE4code4CodeQzvp__SYNTHESIZED__s_SC11VZErrorCodeLeV_set(void* self, void* value);

// Swift method: removeObserver(_:)
void* c_objc_cs_VZGraphicsDisplay_im_removeObserver_(void* self, void* observer);

// Swift property: machineIdentifier
void* c_objc_cs_VZMacPlatformConfiguration_py_machineIdentifier_get(void* self);
void c_objc_cs_VZMacPlatformConfiguration_py_machineIdentifier_set(void* self, void* value);

// Swift property: variableStore
void* c_objc_cs_VZEFIBootLoader_py_variableStore_get(void* self);
void c_objc_cs_VZEFIBootLoader_py_variableStore_set(void* self, void* value);

// Swift method: consoleDevice(_:didOpen:)
void* c_objc_pl_VZVirtioConsoleDeviceDelegate_im_consoleDevice_didOpenPort_(void* self, void* consoleDevice, void* consolePort);

// Swift property: userInfo
void* s_10Foundation21_BridgedStoredNSErrorPAAE8userInfoSDySSypGvp__SYNTHESIZED__s_SC11VZErrorCodeLeV_get(void* self);
void s_10Foundation21_BridgedStoredNSErrorPAAE8userInfoSDySSypGvp__SYNTHESIZED__s_SC11VZErrorCodeLeV_set(void* self, void* value);

// Swift property: minimumSupportedMemorySize
void* c_objc_cs_VZMacOSConfigurationRequirements_py_minimumSupportedMemorySize_get(void* self);
void c_objc_cs_VZMacOSConfigurationRequirements_py_minimumSupportedMemorySize_set(void* self, void* value);

// Swift property: url
void* c_objc_cs_VZFileSerialPortAttachment_py_URL_get(void* self);
void c_objc_cs_VZFileSerialPortAttachment_py_URL_set(void* self, void* value);

// Swift method: reconfigure(sizeInPixels:)
void* c_objc_cs_VZGraphicsDisplay_im_reconfigureWithSizeInPixels_error_(void* self, void* sizeInPixels);

// Swift property: delegate
void* c_objc_cs_VZNetworkBlockDeviceStorageDeviceAttachment_py_delegate_get(void* self);
void c_objc_cs_VZNetworkBlockDeviceStorageDeviceAttachment_py_delegate_set(void* self, void* value);

// Swift property: directorySharingDevices
void* c_objc_cs_VZVirtualMachineConfiguration_py_directorySharingDevices_get(void* self);
void c_objc_cs_VZVirtualMachineConfiguration_py_directorySharingDevices_set(void* self, void* value);

// Swift property: fileHandle
void* c_objc_cs_VZFileHandleNetworkDeviceAttachment_py_fileHandle_get(void* self);
void c_objc_cs_VZFileHandleNetworkDeviceAttachment_py_fileHandle_set(void* self, void* value);

// Swift property: directory
void* c_objc_cs_VZSingleDirectoryShare_py_directory_get(void* self);
void c_objc_cs_VZSingleDirectoryShare_py_directory_set(void* self, void* value);

// Swift property: usbControllers
void* c_objc_cs_VZVirtualMachineConfiguration_py_usbControllers_get(void* self);
void c_objc_cs_VZVirtualMachineConfiguration_py_usbControllers_set(void* self, void* value);

// Swift method: reconfigure(configuration:)
void* c_objc_cs_VZGraphicsDisplay_im_reconfigureWithConfiguration_error_(void* self, void* configuration);

// Swift property: usbDevices
void* c_objc_cs_VZUSBControllerConfiguration_py_usbDevices_get(void* self);
void c_objc_cs_VZUSBControllerConfiguration_py_usbDevices_set(void* self, void* value);

// Swift property: synchronizationMode
void* c_objc_cs_VZNetworkBlockDeviceStorageDeviceAttachment_py_synchronizationMode_get(void* self);
void c_objc_cs_VZNetworkBlockDeviceStorageDeviceAttachment_py_synchronizationMode_set(void* self, void* value);

// Swift property: attachment
void* c_objc_cs_VZStorageDeviceConfiguration_py_attachment_get(void* self);
void c_objc_cs_VZStorageDeviceConfiguration_py_attachment_set(void* self, void* value);

// Swift property: maximumTransmissionUnit
void* c_objc_cs_VZFileHandleNetworkDeviceAttachment_py_maximumTransmissionUnit_get(void* self);
void c_objc_cs_VZFileHandleNetworkDeviceAttachment_py_maximumTransmissionUnit_set(void* self, void* value);

// Swift property: share
void* c_objc_cs_VZVirtioFileSystemDevice_py_share_get(void* self);
void c_objc_cs_VZVirtioFileSystemDevice_py_share_set(void* self, void* value);

// Swift property: machineIdentifier
void* c_objc_cs_VZGenericPlatformConfiguration_py_machineIdentifier_get(void* self);
void c_objc_cs_VZGenericPlatformConfiguration_py_machineIdentifier_set(void* self, void* value);

// Swift property: delegate
void* c_objc_cs_VZVirtualMachine_py_delegate_get(void* self);
void c_objc_cs_VZVirtualMachine_py_delegate_set(void* self, void* value);

// Swift property: tag
void* c_objc_cs_VZVirtioFileSystemDevice_py_tag_get(void* self);
void c_objc_cs_VZVirtioFileSystemDevice_py_tag_set(void* self, void* value);

// Swift property: consoleDevices
void* c_objc_cs_VZVirtualMachineConfiguration_py_consoleDevices_get(void* self);
void c_objc_cs_VZVirtualMachineConfiguration_py_consoleDevices_set(void* self, void* value);

// Swift property: url
void* c_objc_cs_VZEFIVariableStore_py_URL_get(void* self);
void c_objc_cs_VZEFIVariableStore_py_URL_set(void* self, void* value);

// Swift property: delegate
void* c_objc_cs_VZVirtioSocketListener_py_delegate_get(void* self);
void c_objc_cs_VZVirtioSocketListener_py_delegate_set(void* self, void* value);

// Swift property: url
void* c_objc_cs_VZMacOSRestoreImage_py_URL_get(void* self);
void c_objc_cs_VZMacOSRestoreImage_py_URL_set(void* self, void* value);

// Swift property: auxiliaryStorage
void* c_objc_cs_VZMacPlatformConfiguration_py_auxiliaryStorage_get(void* self);
void c_objc_cs_VZMacPlatformConfiguration_py_auxiliaryStorage_set(void* self, void* value);

// Swift property: automaticallyReconfiguresDisplay
void* c_objc_cs_VZVirtualMachineView_py_automaticallyReconfiguresDisplay_get(void* self);
void c_objc_cs_VZVirtualMachineView_py_automaticallyReconfiguresDisplay_set(void* self, void* value);

// Swift property: blockDeviceIdentifier
void* c_objc_cs_VZVirtioBlockDeviceConfiguration_py_blockDeviceIdentifier_get(void* self);
void c_objc_cs_VZVirtioBlockDeviceConfiguration_py_blockDeviceIdentifier_set(void* self, void* value);

// Swift method: pause()
void* c_objc_cs_VZVirtualMachine_im_pauseWithCompletionHandler_(void* self);

// Swift property: graphicsDevices
void* c_objc_cs_VZVirtualMachineConfiguration_py_graphicsDevices_get(void* self);
void c_objc_cs_VZVirtualMachineConfiguration_py_graphicsDevices_set(void* self, void* value);

// Swift property: maximumPortCount
void* c_objc_cs_VZVirtioConsolePortArray_py_maximumPortCount_get(void* self);
void c_objc_cs_VZVirtioConsolePortArray_py_maximumPortCount_set(void* self, void* value);

// Swift property: canStart
void* c_objc_cs_VZVirtualMachine_py_canStart_get(void* self);
void c_objc_cs_VZVirtualMachine_py_canStart_set(void* self, void* value);

// Swift property: isReadOnly
void* c_objc_cs_VZSharedDirectory_py_readOnly_get(void* self);
void c_objc_cs_VZSharedDirectory_py_readOnly_set(void* self, void* value);

// Swift method: guestDidStop(_:)
void* c_objc_pl_VZVirtualMachineDelegate_im_guestDidStopVirtualMachine_(void* self, void* virtualMachine);

// Swift property: append
void* c_objc_cs_VZFileSerialPortAttachment_py_append_get(void* self);
void c_objc_cs_VZFileSerialPortAttachment_py_append_set(void* self, void* value);

// Swift property: sizeInPixels
void* c_objc_cs_VZGraphicsDisplay_py_sizeInPixels_get(void* self);
void c_objc_cs_VZGraphicsDisplay_py_sizeInPixels_set(void* self, void* value);

// Swift property: isUnicastAddress
void* c_objc_cs_VZMACAddress_py_isUnicastAddress_get(void* self);
void c_objc_cs_VZMACAddress_py_isUnicastAddress_set(void* self, void* value);

// Swift method: install()
void* c_objc_cs_VZMacOSInstaller_im_installWithCompletionHandler_(void* self);

// Swift method: listener(_:shouldAcceptNewConnection:from:)
void* c_objc_pl_VZVirtioSocketListenerDelegate_im_listener_shouldAcceptNewConnection_fromSocketDevice_(void* self, void* listener, void* connection, void* socketDevice);

// Swift property: url
void* c_objc_cs_VZNetworkBlockDeviceStorageDeviceAttachment_py_URL_get(void* self);
void c_objc_cs_VZNetworkBlockDeviceStorageDeviceAttachment_py_URL_set(void* self, void* value);

// Swift property: delegate
void* c_objc_cs_VZVirtioConsoleDevice_py_delegate_get(void* self);
void c_objc_cs_VZVirtioConsoleDevice_py_delegate_set(void* self, void* value);

// Swift property: audioDevices
void* c_objc_cs_VZVirtualMachineConfiguration_py_audioDevices_get(void* self);
void c_objc_cs_VZVirtualMachineConfiguration_py_audioDevices_set(void* self, void* value);

// Swift method: virtualMachine(_:didStopWithError:)
void* c_objc_pl_VZVirtualMachineDelegate_im_virtualMachine_didStopWithError_(void* self, void* virtualMachine, void* error);

// Swift property: usbController
void* c_objc_pl_VZUSBDevice_py_usbController_get(void* self);
void c_objc_pl_VZUSBDevice_py_usbController_set(void* self, void* value);

// Swift property: isSupported
void* c_objc_cs_VZMacOSRestoreImage_py_supported_get(void* self);
void c_objc_cs_VZMacOSRestoreImage_py_supported_set(void* self, void* value);

// Swift property: dataRepresentation
void* c_objc_cs_VZMacHardwareModel_py_dataRepresentation_get(void* self);
void c_objc_cs_VZMacHardwareModel_py_dataRepresentation_set(void* self, void* value);

// Swift property: pixelsPerInch
void* c_objc_cs_VZMacGraphicsDisplay_py_pixelsPerInch_get(void* self);
void c_objc_cs_VZMacGraphicsDisplay_py_pixelsPerInch_set(void* self, void* value);

// Swift property: displays
void* c_objc_cs_VZMacGraphicsDeviceConfiguration_py_displays_get(void* self);
void c_objc_cs_VZMacGraphicsDeviceConfiguration_py_displays_set(void* self, void* value);

// Swift property: attachment
void* c_objc_cs_VZNetworkDeviceConfiguration_py_attachment_get(void* self);
void c_objc_cs_VZNetworkDeviceConfiguration_py_attachment_set(void* self, void* value);

// Swift property: state
void* c_objc_cs_VZVirtualMachine_py_state_get(void* self);
void c_objc_cs_VZVirtualMachine_py_state_set(void* self, void* value);

// Swift property: pointingDevices
void* c_objc_cs_VZVirtualMachineConfiguration_py_pointingDevices_get(void* self);
void c_objc_cs_VZVirtualMachineConfiguration_py_pointingDevices_set(void* self, void* value);

// Swift property: scanouts
void* c_objc_cs_VZVirtioGraphicsDeviceConfiguration_py_scanouts_get(void* self);
void c_objc_cs_VZVirtioGraphicsDeviceConfiguration_py_scanouts_set(void* self, void* value);

// Swift property: commandLine
void* c_objc_cs_VZLinuxBootLoader_py_commandLine_get(void* self);
void c_objc_cs_VZLinuxBootLoader_py_commandLine_set(void* self, void* value);

// Swift property: isLocallyAdministeredAddress
void* c_objc_cs_VZMACAddress_py_isLocallyAdministeredAddress_get(void* self);
void c_objc_cs_VZMACAddress_py_isLocallyAdministeredAddress_set(void* self, void* value);

// Swift method: start(options:)
void* c_objc_cs_VZVirtualMachine_im_startWithOptions_completionHandler_(void* self, void* options);

// Swift method: resume()
void* c_objc_cs_VZVirtualMachine_im_resumeWithCompletionHandler_(void* self);

// Swift method: virtualMachine(_:networkDevice:attachmentWasDisconnectedWithError:)
void* c_objc_pl_VZVirtualMachineDelegate_im_virtualMachine_networkDevice_attachmentWasDisconnectedWithError_(void* self, void* virtualMachine, void* networkDevice, void* error);

// Swift property: url
void* c_objc_cs_VZSharedDirectory_py_URL_get(void* self);
void c_objc_cs_VZSharedDirectory_py_URL_set(void* self, void* value);

// Swift property: capturesSystemKeys
void* c_objc_cs_VZVirtualMachineView_py_capturesSystemKeys_get(void* self);
void c_objc_cs_VZVirtualMachineView_py_capturesSystemKeys_set(void* self, void* value);

// Swift property: isUniversallyAdministeredAddress
void* c_objc_cs_VZMACAddress_py_isUniversallyAdministeredAddress_get(void* self);
void c_objc_cs_VZMACAddress_py_isUniversallyAdministeredAddress_set(void* self, void* value);

// Swift property: ethernetAddress
void* c_objc_cs_VZMACAddress_py_ethernetAddress_get(void* self);
void c_objc_cs_VZMACAddress_py_ethernetAddress_set(void* self, void* value);

// Swift property: fileHandleForReading
void* c_objc_cs_VZFileHandleSerialPortAttachment_py_fileHandleForReading_get(void* self);
void c_objc_cs_VZFileHandleSerialPortAttachment_py_fileHandleForReading_set(void* self, void* value);

// Swift property: kernelURL
void* c_objc_cs_VZLinuxBootLoader_py_kernelURL_get(void* self);
void c_objc_cs_VZLinuxBootLoader_py_kernelURL_set(void* self, void* value);

// Swift property: cachingMode
void* c_objc_cs_VZDiskImageStorageDeviceAttachment_py_cachingMode_get(void* self);
void c_objc_cs_VZDiskImageStorageDeviceAttachment_py_cachingMode_set(void* self, void* value);

// Swift property: sharesClipboard
void* c_objc_cs_VZSpiceAgentPortAttachment_py_sharesClipboard_get(void* self);
void c_objc_cs_VZSpiceAgentPortAttachment_py_sharesClipboard_set(void* self, void* value);

// Swift property: interface
void* c_objc_cs_VZBridgedNetworkDeviceAttachment_py_interface_get(void* self);
void c_objc_cs_VZBridgedNetworkDeviceAttachment_py_interface_set(void* self, void* value);

// Swift property: canRequestStop
void* c_objc_cs_VZVirtualMachine_py_canRequestStop_get(void* self);
void c_objc_cs_VZVirtualMachine_py_canRequestStop_set(void* self, void* value);

// Swift property: isReadOnly
void* c_objc_cs_VZDiskImageStorageDeviceAttachment_py_readOnly_get(void* self);
void c_objc_cs_VZDiskImageStorageDeviceAttachment_py_readOnly_set(void* self, void* value);

// Swift property: synchronizationMode
void* c_objc_cs_VZDiskBlockDeviceStorageDeviceAttachment_py_synchronizationMode_get(void* self);
void c_objc_cs_VZDiskBlockDeviceStorageDeviceAttachment_py_synchronizationMode_set(void* self, void* value);

// Swift property: platform
void* c_objc_cs_VZVirtualMachineConfiguration_py_platform_get(void* self);
void c_objc_cs_VZVirtualMachineConfiguration_py_platform_set(void* self, void* value);

// Swift property: uuid
void* c_objc_pl_VZUSBDevice_py_uuid_get(void* self);
void c_objc_pl_VZUSBDevice_py_uuid_set(void* self, void* value);

// Swift property: synchronizationMode
void* c_objc_cs_VZDiskImageStorageDeviceAttachment_py_synchronizationMode_get(void* self);
void c_objc_cs_VZDiskImageStorageDeviceAttachment_py_synchronizationMode_set(void* self, void* value);

// Swift method: start()
void* c_objc_cs_VZVirtualMachine_im_startWithCompletionHandler_(void* self);

// Swift property: attachment
void* c_objc_cs_VZNetworkDevice_py_attachment_get(void* self);
void c_objc_cs_VZNetworkDevice_py_attachment_set(void* self, void* value);

// Swift property: keyboards
void* c_objc_cs_VZVirtualMachineConfiguration_py_keyboards_get(void* self);
void c_objc_cs_VZVirtualMachineConfiguration_py_keyboards_set(void* self, void* value);

// Swift property: attachment
void* c_objc_cs_VZVirtioConsolePort_py_attachment_get(void* self);
void c_objc_cs_VZVirtioConsolePort_py_attachment_set(void* self, void* value);

// Swift property: dataRepresentation
void* c_objc_cs_VZMacMachineIdentifier_py_dataRepresentation_get(void* self);
void c_objc_cs_VZMacMachineIdentifier_py_dataRepresentation_set(void* self, void* value);

// Swift property: usbControllers
void* c_objc_cs_VZVirtualMachine_py_usbControllers_get(void* self);
void c_objc_cs_VZVirtualMachine_py_usbControllers_set(void* self, void* value);

// Swift property: name
void* c_objc_cs_VZVirtioConsolePort_py_name_get(void* self);
void c_objc_cs_VZVirtioConsolePort_py_name_set(void* self, void* value);

// Swift property: widthInPixels
void* c_objc_cs_VZVirtioGraphicsScanoutConfiguration_py_widthInPixels_get(void* self);
void c_objc_cs_VZVirtioGraphicsScanoutConfiguration_py_widthInPixels_set(void* self, void* value);

// Swift method: setSocketListener(_:forPort:)
void* c_objc_cs_VZVirtioSocketDevice_im_setSocketListener_forPort_(void* self, void* listener, void* port);

// Swift property: consoleDevices
void* c_objc_cs_VZVirtualMachine_py_consoleDevices_get(void* self);
void c_objc_cs_VZVirtualMachine_py_consoleDevices_set(void* self, void* value);

// Swift property: isMulticastAddress
void* c_objc_cs_VZMACAddress_py_isMulticastAddress_get(void* self);
void c_objc_cs_VZMACAddress_py_isMulticastAddress_set(void* self, void* value);

// Swift property: buildVersion
void* c_objc_cs_VZMacOSRestoreImage_py_buildVersion_get(void* self);
void c_objc_cs_VZMacOSRestoreImage_py_buildVersion_set(void* self, void* value);

// Swift property: maximumPortCount
void* c_objc_cs_VZVirtioConsolePortConfigurationArray_py_maximumPortCount_get(void* self);
void c_objc_cs_VZVirtioConsolePortConfigurationArray_py_maximumPortCount_set(void* self, void* value);

// Swift property: displays
void* c_objc_cs_VZGraphicsDevice_py_displays_get(void* self);
void c_objc_cs_VZGraphicsDevice_py_displays_set(void* self, void* value);

// Swift property: virtualMachine
void* c_objc_cs_VZVirtualMachineView_py_virtualMachine_get(void* self);
void c_objc_cs_VZVirtualMachineView_py_virtualMachine_set(void* self, void* value);

// Swift property: isSupported
void* c_objc_cs_VZMacHardwareModel_py_supported_get(void* self);
void c_objc_cs_VZMacHardwareModel_py_supported_set(void* self, void* value);

// Swift property: url
void* c_objc_cs_VZDiskImageStorageDeviceAttachment_py_URL_get(void* self);
void c_objc_cs_VZDiskImageStorageDeviceAttachment_py_URL_set(void* self, void* value);

// Swift method: removeSocketListener(forPort:)
void* c_objc_cs_VZVirtioSocketDevice_im_removeSocketListenerForPort_(void* self, void* port);

// Swift property: storageDevices
void* c_objc_cs_VZVirtualMachineConfiguration_py_storageDevices_get(void* self);
void c_objc_cs_VZVirtualMachineConfiguration_py_storageDevices_set(void* self, void* value);

// Swift property: macAddress
void* c_objc_cs_VZNetworkDeviceConfiguration_py_MACAddress_get(void* self);
void c_objc_cs_VZNetworkDeviceConfiguration_py_MACAddress_set(void* self, void* value);

// Swift property: isBroadcastAddress
void* c_objc_cs_VZMACAddress_py_isBroadcastAddress_get(void* self);
void c_objc_cs_VZMACAddress_py_isBroadcastAddress_set(void* self, void* value);

// Swift property: cpuCount
void* c_objc_cs_VZVirtualMachineConfiguration_py_CPUCount_get(void* self);
void c_objc_cs_VZVirtualMachineConfiguration_py_CPUCount_set(void* self, void* value);

// Swift property: canStop
void* c_objc_cs_VZVirtualMachine_py_canStop_get(void* self);
void c_objc_cs_VZVirtualMachine_py_canStop_set(void* self, void* value);

// Swift property: canResume
void* c_objc_cs_VZVirtualMachine_py_canResume_get(void* self);
void c_objc_cs_VZVirtualMachine_py_canResume_set(void* self, void* value);

// Swift property: identifier
void* c_objc_cs_VZBridgedNetworkInterface_py_identifier_get(void* self);
void c_objc_cs_VZBridgedNetworkInterface_py_identifier_set(void* self, void* value);

// Swift property: heightInPixels
void* c_objc_cs_VZVirtioGraphicsScanoutConfiguration_py_heightInPixels_get(void* self);
void c_objc_cs_VZVirtioGraphicsScanoutConfiguration_py_heightInPixels_set(void* self, void* value);

// Swift method: stop()
void* c_objc_cs_VZVirtualMachine_im_stopWithCompletionHandler_(void* self);

// Swift property: string
void* c_objc_cs_VZMACAddress_py_string_get(void* self);
void c_objc_cs_VZMACAddress_py_string_set(void* self, void* value);

// Swift property: canPause
void* c_objc_cs_VZVirtualMachine_py_canPause_get(void* self);
void c_objc_cs_VZVirtualMachine_py_canPause_set(void* self, void* value);

// Swift property: socketDevices
void* c_objc_cs_VZVirtualMachineConfiguration_py_socketDevices_get(void* self);
void c_objc_cs_VZVirtualMachineConfiguration_py_socketDevices_set(void* self, void* value);

// Swift property: memoryBalloonDevices
void* c_objc_cs_VZVirtualMachine_py_memoryBalloonDevices_get(void* self);
void c_objc_cs_VZVirtualMachine_py_memoryBalloonDevices_set(void* self, void* value);

// Swift property: source
void* c_objc_cs_VZVirtioSoundDeviceInputStreamConfiguration_py_source_get(void* self);
void c_objc_cs_VZVirtioSoundDeviceInputStreamConfiguration_py_source_set(void* self, void* value);

// Swift property: bootLoader
void* c_objc_cs_VZVirtualMachineConfiguration_py_bootLoader_get(void* self);
void c_objc_cs_VZVirtualMachineConfiguration_py_bootLoader_set(void* self, void* value);

// Swift property: localizedDisplayName
void* c_objc_cs_VZBridgedNetworkInterface_py_localizedDisplayName_get(void* self);
void c_objc_cs_VZBridgedNetworkInterface_py_localizedDisplayName_set(void* self, void* value);

// Swift property: isConsole
void* c_objc_cs_VZVirtioConsolePortConfiguration_py_isConsole_get(void* self);
void c_objc_cs_VZVirtioConsolePortConfiguration_py_isConsole_set(void* self, void* value);

// Swift property: targetVirtualMachineMemorySize
void* c_objc_cs_VZVirtioTraditionalMemoryBalloonDevice_py_targetVirtualMachineMemorySize_get(void* self);
void c_objc_cs_VZVirtioTraditionalMemoryBalloonDevice_py_targetVirtualMachineMemorySize_set(void* self, void* value);

// Swift method: consoleDevice(_:didClose:)
void* c_objc_pl_VZVirtioConsoleDeviceDelegate_im_consoleDevice_didClosePort_(void* self, void* consoleDevice, void* consolePort);

// Swift property: destinationPort
void* c_objc_cs_VZVirtioSocketConnection_py_destinationPort_get(void* self);
void c_objc_cs_VZVirtioSocketConnection_py_destinationPort_set(void* self, void* value);

// Swift property: operatingSystemVersion
void* c_objc_cs_VZMacOSRestoreImage_py_operatingSystemVersion_get(void* self);
void c_objc_cs_VZMacOSRestoreImage_py_operatingSystemVersion_set(void* self, void* value);

// Swift method: attach(device:)
void* c_objc_cs_VZUSBController_im_attachDevice_completionHandler_(void* self, void* device);

// Swift property: attachment
void* c_objc_cs_VZSerialPortConfiguration_py_attachment_get(void* self);
void c_objc_cs_VZSerialPortConfiguration_py_attachment_set(void* self, void* value);

// Swift property: errorUserInfo
void* s_10Foundation21_BridgedStoredNSErrorPAAE13errorUserInfoSDySSypGvp__SYNTHESIZED__s_SC11VZErrorCodeLeV_get(void* self);
void s_10Foundation21_BridgedStoredNSErrorPAAE13errorUserInfoSDySSypGvp__SYNTHESIZED__s_SC11VZErrorCodeLeV_set(void* self, void* value);

// Swift method: displayDidBeginReconfiguration(_:)
void* c_objc_pl_VZGraphicsDisplayObserver_im_displayDidBeginReconfiguration_(void* self, void* display);

// Swift property: pixelsPerInch
void* c_objc_cs_VZMacGraphicsDisplayConfiguration_py_pixelsPerInch_get(void* self);
void c_objc_cs_VZMacGraphicsDisplayConfiguration_py_pixelsPerInch_set(void* self, void* value);

// Swift property: hardwareModel
void* c_objc_cs_VZMacOSConfigurationRequirements_py_hardwareModel_get(void* self);
void c_objc_cs_VZMacOSConfigurationRequirements_py_hardwareModel_set(void* self, void* value);

// Swift property: serialPorts
void* c_objc_cs_VZVirtualMachineConfiguration_py_serialPorts_get(void* self);
void c_objc_cs_VZVirtualMachineConfiguration_py_serialPorts_set(void* self, void* value);

// Swift property: graphicsDevices
void* c_objc_cs_VZVirtualMachine_py_graphicsDevices_get(void* self);
void c_objc_cs_VZVirtualMachine_py_graphicsDevices_set(void* self, void* value);

// Swift method: displayDidEndReconfiguration(_:)
void* c_objc_pl_VZGraphicsDisplayObserver_im_displayDidEndReconfiguration_(void* self, void* display);

// Swift property: initialRamdiskURL
void* c_objc_cs_VZLinuxBootLoader_py_initialRamdiskURL_get(void* self);
void c_objc_cs_VZLinuxBootLoader_py_initialRamdiskURL_set(void* self, void* value);

// Swift property: mostFeaturefulSupportedConfiguration
void* c_objc_cs_VZMacOSRestoreImage_py_mostFeaturefulSupportedConfiguration_get(void* self);
void c_objc_cs_VZMacOSRestoreImage_py_mostFeaturefulSupportedConfiguration_set(void* self, void* value);

// Swift property: minimumSupportedCPUCount
void* c_objc_cs_VZMacOSConfigurationRequirements_py_minimumSupportedCPUCount_get(void* self);
void c_objc_cs_VZMacOSConfigurationRequirements_py_minimumSupportedCPUCount_set(void* self, void* value);

// Swift property: memorySize
void* c_objc_cs_VZVirtualMachineConfiguration_py_memorySize_get(void* self);
void c_objc_cs_VZVirtualMachineConfiguration_py_memorySize_set(void* self, void* value);

// Swift property: errorCode
void* s_10Foundation21_BridgedStoredNSErrorPAAE9errorCodeSivp__SYNTHESIZED__s_SC11VZErrorCodeLeV_get(void* self);
void s_10Foundation21_BridgedStoredNSErrorPAAE9errorCodeSivp__SYNTHESIZED__s_SC11VZErrorCodeLeV_set(void* self, void* value);

// Swift property: name
void* c_objc_cs_VZVirtioConsolePortConfiguration_py_name_get(void* self);
void c_objc_cs_VZVirtioConsolePortConfiguration_py_name_set(void* self, void* value);

// Swift property: fileDescriptor
void* c_objc_cs_VZVirtioSocketConnection_py_fileDescriptor_get(void* self);
void c_objc_cs_VZVirtioSocketConnection_py_fileDescriptor_set(void* self, void* value);

// Swift property: heightInPixels
void* c_objc_cs_VZMacGraphicsDisplayConfiguration_py_heightInPixels_get(void* self);
void c_objc_cs_VZMacGraphicsDisplayConfiguration_py_heightInPixels_set(void* self, void* value);

// Swift property: sink
void* c_objc_cs_VZVirtioSoundDeviceOutputStreamConfiguration_py_sink_get(void* self);
void c_objc_cs_VZVirtioSoundDeviceOutputStreamConfiguration_py_sink_set(void* self, void* value);

// Swift property: restoreImageURL
void* c_objc_cs_VZMacOSInstaller_py_restoreImageURL_get(void* self);
void c_objc_cs_VZMacOSInstaller_py_restoreImageURL_set(void* self, void* value);

// Swift method: saveMachineStateTo(url:)
void* c_objc_cs_VZVirtualMachine_im_saveMachineStateToURL_completionHandler_(void* self, void* saveFileURL);

// Swift property: isNestedVirtualizationEnabled
void* c_objc_cs_VZGenericPlatformConfiguration_py_nestedVirtualizationEnabled_get(void* self);
void c_objc_cs_VZGenericPlatformConfiguration_py_nestedVirtualizationEnabled_set(void* self, void* value);

// Swift property: socketDevices
void* c_objc_cs_VZVirtualMachine_py_socketDevices_get(void* self);
void c_objc_cs_VZVirtualMachine_py_socketDevices_set(void* self, void* value);

// Swift property: networkDevices
void* c_objc_cs_VZVirtualMachineConfiguration_py_networkDevices_get(void* self);
void c_objc_cs_VZVirtualMachineConfiguration_py_networkDevices_set(void* self, void* value);

// Swift method: connect(toPort:)
void* c_objc_cs_VZVirtioSocketDevice_im_connectToPort_completionHandler_(void* self, void* port);

// Swift method: attachment(_:didEncounterError:)
void* c_objc_pl_VZNetworkBlockDeviceStorageDeviceAttachmentDelegate_im_attachment_didEncounterError_(void* self, void* attachment, void* error);

// Swift property: directorySharingDevices
void* c_objc_cs_VZVirtualMachine_py_directorySharingDevices_get(void* self);
void c_objc_cs_VZVirtualMachine_py_directorySharingDevices_set(void* self, void* value);

// Swift method: close()
void* c_objc_cs_VZVirtioSocketConnection_im_close(void* self);

// Swift property: fileHandleForWriting
void* c_objc_cs_VZFileHandleSerialPortAttachment_py_fileHandleForWriting_get(void* self);
void c_objc_cs_VZFileHandleSerialPortAttachment_py_fileHandleForWriting_set(void* self, void* value);

// Swift property: fileHandle
void* c_objc_cs_VZDiskBlockDeviceStorageDeviceAttachment_py_fileHandle_get(void* self);
void c_objc_cs_VZDiskBlockDeviceStorageDeviceAttachment_py_fileHandle_set(void* self, void* value);

// Swift property: uuid
void* c_objc_pl_VZUSBDeviceConfiguration_py_uuid_get(void* self);
void c_objc_pl_VZUSBDeviceConfiguration_py_uuid_set(void* self, void* value);

// Swift property: dataRepresentation
void* c_objc_cs_VZGenericMachineIdentifier_py_dataRepresentation_get(void* self);
void c_objc_cs_VZGenericMachineIdentifier_py_dataRepresentation_set(void* self, void* value);

// Swift property: networkDevices
void* c_objc_cs_VZVirtualMachine_py_networkDevices_get(void* self);
void c_objc_cs_VZVirtualMachine_py_networkDevices_set(void* self, void* value);

// Swift property: directories
void* c_objc_cs_VZMultipleDirectoryShare_py_directories_get(void* self);
void c_objc_cs_VZMultipleDirectoryShare_py_directories_set(void* self, void* value);

// Swift property: progress
void* c_objc_cs_VZMacOSInstaller_py_progress_get(void* self);
void c_objc_cs_VZMacOSInstaller_py_progress_set(void* self, void* value);

// Swift property: startUpFromMacOSRecovery
void* c_objc_cs_VZMacOSVirtualMachineStartOptions_py_startUpFromMacOSRecovery_get(void* self);
void c_objc_cs_VZMacOSVirtualMachineStartOptions_py_startUpFromMacOSRecovery_set(void* self, void* value);

// Swift property: virtualMachine
void* c_objc_cs_VZMacOSInstaller_py_virtualMachine_get(void* self);
void c_objc_cs_VZMacOSInstaller_py_virtualMachine_set(void* self, void* value);

// Swift method: requestStop()
void* c_objc_cs_VZVirtualMachine_im_requestStopWithError_(void* self);

// Swift property: sourcePort
void* c_objc_cs_VZVirtioSocketConnection_py_sourcePort_get(void* self);
void c_objc_cs_VZVirtioSocketConnection_py_sourcePort_set(void* self, void* value);

// Swift property: memoryBalloonDevices
void* c_objc_cs_VZVirtualMachineConfiguration_py_memoryBalloonDevices_get(void* self);
void c_objc_cs_VZVirtualMachineConfiguration_py_memoryBalloonDevices_set(void* self, void* value);

// Swift property: url
void* c_objc_cs_VZMacAuxiliaryStorage_py_URL_get(void* self);
void c_objc_cs_VZMacAuxiliaryStorage_py_URL_set(void* self, void* value);

// Swift property: isReadOnly
void* c_objc_cs_VZDiskBlockDeviceStorageDeviceAttachment_py_readOnly_get(void* self);
void c_objc_cs_VZDiskBlockDeviceStorageDeviceAttachment_py_readOnly_set(void* self, void* value);

// Swift method: attachmentWasConnected(_:)
void* c_objc_pl_VZNetworkBlockDeviceStorageDeviceAttachmentDelegate_im_attachmentWasConnected_(void* self, void* attachment);


#ifdef __cplusplus
}
#endif

#endif // VIRTUALIZATION_SHIM_H
