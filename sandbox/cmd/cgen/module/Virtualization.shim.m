
#import <Foundation/Foundation.h>
#import <objc/message.h>
#import <objc/runtime.h>
#import <Virtualization/Virtualization.h>

#ifdef __cplusplus
extern "C" {
#endif
// Shim for Swift method: virtualMachine(_:didStopWithError:)
void* c_objc_pl_VZVirtualMachineDelegate_im_virtualMachine_didStopWithError_(void* self, void* virtualMachine, void* error) {
    id obj = (__bridge id)self;
    // cast objc_msgSend to a function pointer matching (id,SEL, id, id, id) -> void*
    typedef void* (*MsgFn)(id, SEL, id, id, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj,
               sel_getUid("virtualMachine(_:didStopWithError:):"),
               virtualMachine, error);
    return (__bridge_retained void*)rv;
}


// Shim for Swift method: reconfigure(configuration:)
void* c_objc_cs_VZGraphicsDisplay_im_reconfigureWithConfiguration_error_(void* self, void* configuration) {
    id obj = (__bridge id)self;
    // cast objc_msgSend to a function pointer matching (id,SEL, id, id) -> void*
    typedef void* (*MsgFn)(id, SEL, id, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj,
               sel_getUid("reconfigure(configuration:):"),
               configuration);
    return (__bridge_retained void*)rv;
}


// Shim for Swift method: stop(completionHandler:)
void* c_objc_cs_VZVirtualMachine_im_stopWithCompletionHandler_(void* self, void* completionHandler) {
    id obj = (__bridge id)self;
    // cast objc_msgSend to a function pointer matching (id,SEL, id, id) -> void*
    typedef void* (*MsgFn)(id, SEL, id, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj,
               sel_getUid("stop(completionHandler:):"),
               completionHandler);
    return (__bridge_retained void*)rv;
}


// Shim for Swift method: consoleDevice(_:didOpen:)
void* c_objc_pl_VZVirtioConsoleDeviceDelegate_im_consoleDevice_didOpenPort_(void* self, void* consoleDevice, void* consolePort) {
    id obj = (__bridge id)self;
    // cast objc_msgSend to a function pointer matching (id,SEL, id, id, id) -> void*
    typedef void* (*MsgFn)(id, SEL, id, id, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj,
               sel_getUid("consoleDevice(_:didOpen:):"),
               consoleDevice, consolePort);
    return (__bridge_retained void*)rv;
}


// Shim for Swift method: stop()
void* c_objc_cs_VZVirtualMachine_im_stopWithCompletionHandler_(void* self) {
    id obj = (__bridge id)self;
    // cast objc_msgSend to a function pointer matching (id,SEL) -> void*
    typedef void* (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    
    id rv = fn(obj, sel_getUid("stop()"));
    return (__bridge_retained void*)rv;
}


// Shim for Swift method: addObserver(_:)
void* c_objc_cs_VZGraphicsDisplay_im_addObserver_(void* self, void* observer) {
    id obj = (__bridge id)self;
    // cast objc_msgSend to a function pointer matching (id,SEL, id, id) -> void*
    typedef void* (*MsgFn)(id, SEL, id, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj,
               sel_getUid("addObserver(_:):"),
               observer);
    return (__bridge_retained void*)rv;
}


// Shim for Swift method: attachmentWasConnected(_:)
void* c_objc_pl_VZNetworkBlockDeviceStorageDeviceAttachmentDelegate_im_attachmentWasConnected_(void* self, void* attachment) {
    id obj = (__bridge id)self;
    // cast objc_msgSend to a function pointer matching (id,SEL, id, id) -> void*
    typedef void* (*MsgFn)(id, SEL, id, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj,
               sel_getUid("attachmentWasConnected(_:):"),
               attachment);
    return (__bridge_retained void*)rv;
}


// Shim for Swift method: virtualMachine(_:networkDevice:attachmentWasDisconnectedWithError:)
void* c_objc_pl_VZVirtualMachineDelegate_im_virtualMachine_networkDevice_attachmentWasDisconnectedWithError_(void* self, void* virtualMachine, void* networkDevice, void* error) {
    id obj = (__bridge id)self;
    // cast objc_msgSend to a function pointer matching (id,SEL, id, id, id, id) -> void*
    typedef void* (*MsgFn)(id, SEL, id, id, id, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj,
               sel_getUid("virtualMachine(_:networkDevice:attachmentWasDisconnectedWithError:):"),
               virtualMachine, networkDevice, error);
    return (__bridge_retained void*)rv;
}


// Shim for Swift method: close()
void* c_objc_cs_VZVirtioSocketConnection_im_close(void* self) {
    id obj = (__bridge id)self;
    // cast objc_msgSend to a function pointer matching (id,SEL) -> void*
    typedef void* (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    
    id rv = fn(obj, sel_getUid("close()"));
    return (__bridge_retained void*)rv;
}


// Shim for Swift method: attachment(_:didEncounterError:)
void* c_objc_pl_VZNetworkBlockDeviceStorageDeviceAttachmentDelegate_im_attachment_didEncounterError_(void* self, void* attachment, void* error) {
    id obj = (__bridge id)self;
    // cast objc_msgSend to a function pointer matching (id,SEL, id, id, id) -> void*
    typedef void* (*MsgFn)(id, SEL, id, id, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj,
               sel_getUid("attachment(_:didEncounterError:):"),
               attachment, error);
    return (__bridge_retained void*)rv;
}


// Shim for Swift method: reconfigure(sizeInPixels:)
void* c_objc_cs_VZGraphicsDisplay_im_reconfigureWithSizeInPixels_error_(void* self, void* sizeInPixels) {
    id obj = (__bridge id)self;
    // cast objc_msgSend to a function pointer matching (id,SEL, id, id) -> void*
    typedef void* (*MsgFn)(id, SEL, id, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj,
               sel_getUid("reconfigure(sizeInPixels:):"),
               sizeInPixels);
    return (__bridge_retained void*)rv;
}


// Shim for Swift method: requestStop()
void* c_objc_cs_VZVirtualMachine_im_requestStopWithError_(void* self) {
    id obj = (__bridge id)self;
    // cast objc_msgSend to a function pointer matching (id,SEL) -> void*
    typedef void* (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    
    id rv = fn(obj, sel_getUid("requestStop()"));
    return (__bridge_retained void*)rv;
}


// Shim for Swift method: displayDidEndReconfiguration(_:)
void* c_objc_pl_VZGraphicsDisplayObserver_im_displayDidEndReconfiguration_(void* self, void* display) {
    id obj = (__bridge id)self;
    // cast objc_msgSend to a function pointer matching (id,SEL, id, id) -> void*
    typedef void* (*MsgFn)(id, SEL, id, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj,
               sel_getUid("displayDidEndReconfiguration(_:):"),
               display);
    return (__bridge_retained void*)rv;
}


// Shim for Swift method: listener(_:shouldAcceptNewConnection:from:)
void* c_objc_pl_VZVirtioSocketListenerDelegate_im_listener_shouldAcceptNewConnection_fromSocketDevice_(void* self, void* listener, void* connection, void* socketDevice) {
    id obj = (__bridge id)self;
    // cast objc_msgSend to a function pointer matching (id,SEL, id, id, id, id) -> void*
    typedef void* (*MsgFn)(id, SEL, id, id, id, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj,
               sel_getUid("listener(_:shouldAcceptNewConnection:from:):"),
               listener, connection, socketDevice);
    return (__bridge_retained void*)rv;
}


// Shim for Swift method: displayDidBeginReconfiguration(_:)
void* c_objc_pl_VZGraphicsDisplayObserver_im_displayDidBeginReconfiguration_(void* self, void* display) {
    id obj = (__bridge id)self;
    // cast objc_msgSend to a function pointer matching (id,SEL, id, id) -> void*
    typedef void* (*MsgFn)(id, SEL, id, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj,
               sel_getUid("displayDidBeginReconfiguration(_:):"),
               display);
    return (__bridge_retained void*)rv;
}


// Shim for Swift method: start()
void* c_objc_cs_VZVirtualMachine_im_startWithCompletionHandler_(void* self) {
    id obj = (__bridge id)self;
    // cast objc_msgSend to a function pointer matching (id,SEL) -> void*
    typedef void* (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    
    id rv = fn(obj, sel_getUid("start()"));
    return (__bridge_retained void*)rv;
}


// Shim for Swift method: saveMachineStateTo(url:)
void* c_objc_cs_VZVirtualMachine_im_saveMachineStateToURL_completionHandler_(void* self, void* saveFileURL) {
    id obj = (__bridge id)self;
    // cast objc_msgSend to a function pointer matching (id,SEL, id, id) -> void*
    typedef void* (*MsgFn)(id, SEL, id, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj,
               sel_getUid("saveMachineStateTo(url:):"),
               saveFileURL);
    return (__bridge_retained void*)rv;
}


// Shim for Swift method: removeObserver(_:)
void* c_objc_cs_VZGraphicsDisplay_im_removeObserver_(void* self, void* observer) {
    id obj = (__bridge id)self;
    // cast objc_msgSend to a function pointer matching (id,SEL, id, id) -> void*
    typedef void* (*MsgFn)(id, SEL, id, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj,
               sel_getUid("removeObserver(_:):"),
               observer);
    return (__bridge_retained void*)rv;
}


// Shim for Swift method: attach(device:)
void* c_objc_cs_VZUSBController_im_attachDevice_completionHandler_(void* self, void* device) {
    id obj = (__bridge id)self;
    // cast objc_msgSend to a function pointer matching (id,SEL, id, id) -> void*
    typedef void* (*MsgFn)(id, SEL, id, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj,
               sel_getUid("attach(device:):"),
               device);
    return (__bridge_retained void*)rv;
}


// Shim for Swift method: attach(device:completionHandler:)
void* c_objc_cs_VZUSBController_im_attachDevice_completionHandler_(void* self, void* device, void* completionHandler) {
    id obj = (__bridge id)self;
    // cast objc_msgSend to a function pointer matching (id,SEL, id, id, id) -> void*
    typedef void* (*MsgFn)(id, SEL, id, id, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj,
               sel_getUid("attach(device:completionHandler:):"),
               device, completionHandler);
    return (__bridge_retained void*)rv;
}


// Shim for Swift method: start(options:)
void* c_objc_cs_VZVirtualMachine_im_startWithOptions_completionHandler_(void* self, void* options) {
    id obj = (__bridge id)self;
    // cast objc_msgSend to a function pointer matching (id,SEL, id, id) -> void*
    typedef void* (*MsgFn)(id, SEL, id, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj,
               sel_getUid("start(options:):"),
               options);
    return (__bridge_retained void*)rv;
}


// Shim for Swift method: start(options:completionHandler:)
void* c_objc_cs_VZVirtualMachine_im_startWithOptions_completionHandler_(void* self, void* options, void* completionHandler) {
    id obj = (__bridge id)self;
    // cast objc_msgSend to a function pointer matching (id,SEL, id, id, id) -> void*
    typedef void* (*MsgFn)(id, SEL, id, id, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj,
               sel_getUid("start(options:completionHandler:):"),
               options, completionHandler);
    return (__bridge_retained void*)rv;
}


// Shim for Swift method: detach(device:)
void* c_objc_cs_VZUSBController_im_detachDevice_completionHandler_(void* self, void* device) {
    id obj = (__bridge id)self;
    // cast objc_msgSend to a function pointer matching (id,SEL, id, id) -> void*
    typedef void* (*MsgFn)(id, SEL, id, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj,
               sel_getUid("detach(device:):"),
               device);
    return (__bridge_retained void*)rv;
}


// Shim for Swift method: saveMachineStateTo(url:completionHandler:)
void* c_objc_cs_VZVirtualMachine_im_saveMachineStateToURL_completionHandler_(void* self, void* saveFileURL, void* completionHandler) {
    id obj = (__bridge id)self;
    // cast objc_msgSend to a function pointer matching (id,SEL, id, id, id) -> void*
    typedef void* (*MsgFn)(id, SEL, id, id, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj,
               sel_getUid("saveMachineStateTo(url:completionHandler:):"),
               saveFileURL, completionHandler);
    return (__bridge_retained void*)rv;
}


// Shim for Swift method: validateSaveRestoreSupport()
void* c_objc_cs_VZVirtualMachineConfiguration_im_validateSaveRestoreSupportWithError_(void* self) {
    id obj = (__bridge id)self;
    // cast objc_msgSend to a function pointer matching (id,SEL) -> void*
    typedef void* (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    
    id rv = fn(obj, sel_getUid("validateSaveRestoreSupport()"));
    return (__bridge_retained void*)rv;
}


// Shim for Swift method: removeSocketListener(forPort:)
void* c_objc_cs_VZVirtioSocketDevice_im_removeSocketListenerForPort_(void* self, void* port) {
    id obj = (__bridge id)self;
    // cast objc_msgSend to a function pointer matching (id,SEL, id, id) -> void*
    typedef void* (*MsgFn)(id, SEL, id, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj,
               sel_getUid("removeSocketListener(forPort:):"),
               port);
    return (__bridge_retained void*)rv;
}


// Shim for Swift method: setSocketListener(_:forPort:)
void* c_objc_cs_VZVirtioSocketDevice_im_setSocketListener_forPort_(void* self, void* listener, void* port) {
    id obj = (__bridge id)self;
    // cast objc_msgSend to a function pointer matching (id,SEL, id, id, id) -> void*
    typedef void* (*MsgFn)(id, SEL, id, id, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj,
               sel_getUid("setSocketListener(_:forPort:):"),
               listener, port);
    return (__bridge_retained void*)rv;
}


// Shim for Swift method: validate()
void* c_objc_cs_VZVirtualMachineConfiguration_im_validateWithError_(void* self) {
    id obj = (__bridge id)self;
    // cast objc_msgSend to a function pointer matching (id,SEL) -> void*
    typedef void* (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    
    id rv = fn(obj, sel_getUid("validate()"));
    return (__bridge_retained void*)rv;
}


// Shim for Swift method: restoreMachineStateFrom(url:completionHandler:)
void* c_objc_cs_VZVirtualMachine_im_restoreMachineStateFromURL_completionHandler_(void* self, void* saveFileURL, void* completionHandler) {
    id obj = (__bridge id)self;
    // cast objc_msgSend to a function pointer matching (id,SEL, id, id, id) -> void*
    typedef void* (*MsgFn)(id, SEL, id, id, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj,
               sel_getUid("restoreMachineStateFrom(url:completionHandler:):"),
               saveFileURL, completionHandler);
    return (__bridge_retained void*)rv;
}


// Shim for Swift method: detach(device:completionHandler:)
void* c_objc_cs_VZUSBController_im_detachDevice_completionHandler_(void* self, void* device, void* completionHandler) {
    id obj = (__bridge id)self;
    // cast objc_msgSend to a function pointer matching (id,SEL, id, id, id) -> void*
    typedef void* (*MsgFn)(id, SEL, id, id, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj,
               sel_getUid("detach(device:completionHandler:):"),
               device, completionHandler);
    return (__bridge_retained void*)rv;
}


// Shim for Swift method: restoreMachineStateFrom(url:)
void* c_objc_cs_VZVirtualMachine_im_restoreMachineStateFromURL_completionHandler_(void* self, void* saveFileURL) {
    id obj = (__bridge id)self;
    // cast objc_msgSend to a function pointer matching (id,SEL, id, id) -> void*
    typedef void* (*MsgFn)(id, SEL, id, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj,
               sel_getUid("restoreMachineStateFrom(url:):"),
               saveFileURL);
    return (__bridge_retained void*)rv;
}


// Shim for Swift method: guestDidStop(_:)
void* c_objc_pl_VZVirtualMachineDelegate_im_guestDidStopVirtualMachine_(void* self, void* virtualMachine) {
    id obj = (__bridge id)self;
    // cast objc_msgSend to a function pointer matching (id,SEL, id, id) -> void*
    typedef void* (*MsgFn)(id, SEL, id, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj,
               sel_getUid("guestDidStop(_:):"),
               virtualMachine);
    return (__bridge_retained void*)rv;
}


// Shim for Swift method: resume()
void* c_objc_cs_VZVirtualMachine_im_resumeWithCompletionHandler_(void* self) {
    id obj = (__bridge id)self;
    // cast objc_msgSend to a function pointer matching (id,SEL) -> void*
    typedef void* (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    
    id rv = fn(obj, sel_getUid("resume()"));
    return (__bridge_retained void*)rv;
}


// Shim for Swift method: connect(toPort:)
void* c_objc_cs_VZVirtioSocketDevice_im_connectToPort_completionHandler_(void* self, void* port) {
    id obj = (__bridge id)self;
    // cast objc_msgSend to a function pointer matching (id,SEL, id, id) -> void*
    typedef void* (*MsgFn)(id, SEL, id, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj,
               sel_getUid("connect(toPort:):"),
               port);
    return (__bridge_retained void*)rv;
}


// Shim for Swift method: consoleDevice(_:didClose:)
void* c_objc_pl_VZVirtioConsoleDeviceDelegate_im_consoleDevice_didClosePort_(void* self, void* consoleDevice, void* consolePort) {
    id obj = (__bridge id)self;
    // cast objc_msgSend to a function pointer matching (id,SEL, id, id, id) -> void*
    typedef void* (*MsgFn)(id, SEL, id, id, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj,
               sel_getUid("consoleDevice(_:didClose:):"),
               consoleDevice, consolePort);
    return (__bridge_retained void*)rv;
}


// Shim for Swift method: install()
void* c_objc_cs_VZMacOSInstaller_im_installWithCompletionHandler_(void* self) {
    id obj = (__bridge id)self;
    // cast objc_msgSend to a function pointer matching (id,SEL) -> void*
    typedef void* (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    
    id rv = fn(obj, sel_getUid("install()"));
    return (__bridge_retained void*)rv;
}


// Shim for Swift method: pause()
void* c_objc_cs_VZVirtualMachine_im_pauseWithCompletionHandler_(void* self) {
    id obj = (__bridge id)self;
    // cast objc_msgSend to a function pointer matching (id,SEL) -> void*
    typedef void* (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    
    id rv = fn(obj, sel_getUid("pause()"));
    return (__bridge_retained void*)rv;
}



#ifdef __cplusplus
}
#endif
