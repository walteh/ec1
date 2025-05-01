package vf

// "C"

// // return
// // }
// var libobjc uintptr

// // —————————————————————————
// // 1) In init(), dlopen the Objective-C runtime, Foundation, and Virtualization
// // —————————————————————————
// func init() {
// 	var err error
// 	libobjc, err = purego.Dlopen(
// 		"/usr/lib/libobjc.A.dylib",
// 		purego.RTLD_LAZY|purego.RTLD_GLOBAL,
// 	)
// 	if err != nil {
// 		panic(err)
// 	}
// 	// so NSArray, NSNumber, etc. are available
// 	if _, err := purego.Dlopen("/System/Library/Frameworks/Foundation.framework/Foundation", purego.RTLD_LAZY|purego.RTLD_GLOBAL); err != nil {
// 		panic(err)
// 	}
// 	// and finally the Virtualization framework itself
// 	if _, err := purego.Dlopen("/System/Library/Frameworks/Virtualization.framework/Virtualization", purego.RTLD_LAZY|purego.RTLD_GLOBAL); err != nil {
// 		panic(err)
// 	}

// 	// cache selectors
// 	sel_memoryBalloonDevices = objc.RegisterName("memoryBalloonDevices")
// 	sel_count = objc.RegisterName("count")
// 	sel_objectAtIndex = objc.RegisterName("objectAtIndex:")
// 	sel_targetVirtualMachineMemorySize = objc.RegisterName("targetVirtualMachineMemorySize")
// 	sel_setTargetVirtualMachineMemorySize = objc.RegisterName("setTargetVirtualMachineMemorySize:")
// 	sel_getTargetVirtualMachineMemorySize = objc.RegisterName("getTargetVirtualMachineMemorySize")

// }

// var object_getClassName func(obj objc.ID) string

// func init() {
// 	purego.RegisterLibFunc(
// 		&object_getClassName,
// 		libobjc,
// 		"object_getClassName",
// 	)
// }

// // 1) Returns an array of Ivar; outCount is number of ivars
// var class_copyIvarList func(objc.Class, *uint32) unsafe.Pointer

// // 2) Get the ivar’s name as a C string → Go string
// var ivar_getName func(objc.Ivar) string

// // 3) Get the byte‐offset of the ivar in the object struct
// var ivar_getOffset func(objc.Ivar) uintptr

// // 4) Read the raw ivar value (an object pointer)
// var object_getIvar func(objc.ID, objc.Ivar) objc.ID

// var (
// 	selValueForKey           = objc.RegisterName("valueForKey:")
// 	selUnsignedLongLongValue = objc.RegisterName("unsignedLongLongValue")
// )

// func init() {
// 	purego.RegisterLibFunc(&class_copyIvarList, libobjc, "class_copyIvarList") //  [oai_citation:0‡Apple Developer](https://developer.apple.com/documentation/objectivec/1418910-class_copyivarlist?utm_source=chatgpt.com) [oai_citation:1‡Apple Developer](https://developer.apple.com/documentation/objectivec/class_copyivarlist%28_%3A_%3A%29?language=objc&utm_source=chatgpt.com)
// 	purego.RegisterLibFunc(&ivar_getName, libobjc, "ivar_getName")             //  [oai_citation:2‡Apple Developer](https://developer.apple.com/documentation/objectivec/1418922-ivar_getname?utm_source=chatgpt.com) [oai_citation:3‡Apple Developer](https://developer.apple.com/documentation/objectivec/ivar_getname%28_%3A%29?language=objc&utm_source=chatgpt.com)
// 	purego.RegisterLibFunc(&ivar_getOffset, libobjc, "ivar_getOffset")         //  [oai_citation:4‡GitHub](https://github.com/gnustep/libobjc2/blob/master/objc/runtime.h?utm_source=chatgpt.com)
// 	purego.RegisterLibFunc(&object_getIvar, libobjc, "object_getIvar")         //  [oai_citation:5‡Stack Overflow](https://stackoverflow.com/questions/1216262/handling-the-return-value-of-object-getivarid-object-ivar-ivar?utm_source=chatgpt.com)
// }

// // —————————————————————————
// // 2) Grab your classes & selectors
// // —————————————————————————
// var (
// 	vzVirtioTraditionalMemoryBalloonDeviceClass objc.Class

// 	// selectors (filled in init)
// 	sel_memoryBalloonDevices              objc.SEL
// 	sel_count                             objc.SEL
// 	sel_objectAtIndex                     objc.SEL
// 	sel_targetVirtualMachineMemorySize    objc.SEL
// 	sel_setTargetVirtualMachineMemorySize objc.SEL
// 	sel_getTargetVirtualMachineMemorySize objc.SEL
// 	ivar_targetVirtualMachineMemorySize   objc.Ivar
// )

// func init() {
// 	vzVirtioTraditionalMemoryBalloonDeviceClass = objc.GetClass("VZVirtioTraditionalMemoryBalloonDevice")
// 	ivar_targetVirtualMachineMemorySize = vzVirtioTraditionalMemoryBalloonDeviceClass.InstanceVariable("targetVirtualMachineMemorySize")
// }

// // —————————————————————————
// // 3) Enumerate the balloon devices
// // —————————————————————————
// type VirtioTraditionalMemoryBalloonDevice struct {
// 	id objc.ID
// }

// func (v *VirtualMachine) objcPtr() unsafe.Pointer {
// 	return hack.GetUnexportedFieldOf(v.vzvm, "_ptr").(unsafe.Pointer)
// }

// func (v *VirtualMachine) MemoryBalloonDevices() ([]*VirtioTraditionalMemoryBalloonDevice, error) {
// 	ptr := v.objcPtr()
// 	if ptr == nil {
// 		return nil, fmt.Errorf("invalid VM pointer")
// 	}
// 	arrID := objc.ID(ptr).Send(sel_memoryBalloonDevices)
// 	if arrID == 0 {
// 		return nil, fmt.Errorf("memoryBalloonDevices returned nil")
// 	}

// 	c := int(objc.Send[uint64](arrID, sel_count)) // count returns a primitive NSUInteger => Go uint64
// 	out := make([]*VirtioTraditionalMemoryBalloonDevice, c)
// 	for i := 0; i < c; i++ {
// 		dev := objc.ID(arrID).Send(sel_objectAtIndex, i)
// 		if dev == 0 {
// 			return nil, fmt.Errorf("objectAtIndex(%d) returned nil", i)
// 		}
// 		out[i] = &VirtioTraditionalMemoryBalloonDevice{id: dev}
// 	}
// 	return out, nil
// }

// // —————————————————————————
// // 4) Getter & setter for the primitive unsigned long long property
// // —————————————————————————
// func (d *VirtioTraditionalMemoryBalloonDevice) SetTargetVirtualMachineMemorySize(size uint64) error {
// 	if d.id == 0 {
// 		return fmt.Errorf("nil balloon device")
// 	}
// 	fmt.Println("d.id type", object_getClassName(d.id))
// 	// use the _setter_ SEL and pass the raw uint64
// 	objc.Send[uintptr](d.id, sel_setTargetVirtualMachineMemorySize, size)
// 	return nil
// }

// var (
// 	class_getInstanceVariable func(objc.Class, string) objc.Ivar
// 	// ivar_getOffset            func(objc.Ivar) uintptr
// )

// func init() {
// 	libobjc, err := purego.Dlopen("/usr/lib/libobjc.A.dylib",
// 		purego.RTLD_LAZY|purego.RTLD_GLOBAL)
// 	if err != nil {
// 		panic(err)
// 	}
// 	purego.RegisterLibFunc(&class_getInstanceVariable, libobjc, "class_getInstanceVariable") //  [oai_citation:10‡Apple Developer](https://developer.apple.com/documentation/objectivec/1418643-class_getinstancevariable?utm_source=chatgpt.com)
// 	purego.RegisterLibFunc(&ivar_getOffset, libobjc, "ivar_getOffset")                       //  [oai_citation:11‡GitHub](https://github.com/gnustep/libobjc2/blob/master/objc/runtime.h?utm_source=chatgpt.com)
// }

// // func (d *VirtioTraditionalMemoryBalloonDevice) GetTargetVirtualMachineMemorySize() (uint64, error) {
// // 	if d.id == 0 {
// // 		return 0, fmt.Errorf("nil balloon device")
// // 	}

// // 	fmt.Println("d.id type", object_getClassName(d.id))
// // 	// 1) Ask for the boxed NSNumber
// // 	numObj := d.id.Send(selValueForKey, "targetVirtualMachineMemorySize")
// // 	if numObj == 0 {
// // 		return 0, fmt.Errorf("valueForKey: returned nil")
// // 	}

// // 	fmt.Println("numObj", numObj)
// // 	// 2) Unbox to uint64
// // 	raw := objc.ID(numObj).Send(selUnsignedLongLongValue)
// // 	fmt.Println("raw", raw)

// // 	// 1) Find the Ivar by name
// // 	iv := class_getInstanceVariable(d.id.Class(), "targetVirtualMachineMemorySize")
// // 	if iv == 0 {
// // 		return 0, fmt.Errorf("ivar not found")
// // 	}

// // 	// 2) Compute its byte‐offset in the object
// // 	offset := ivar_getOffset(iv)

// // 	// 3) Read the 64-bit value at that address
// // 	basePtr := unsafe.Pointer(uintptr(d.id))
// // 	fieldPtr := unsafe.Pointer(uintptr(basePtr) + offset)
// // 	val := *(*uint64)(fieldPtr)

// // 	// ret := d.id.Send(sel_targetVirtualMachineMemorySize)

// // 	// call the _getter_ SEL and receive a raw uint64 back
// // 	return uint64(val), nil
// // }

// // func init() {
// // 	var err error
// // 	// 1) Load the Objective-C runtime
// // 	libobjc, err = purego.Dlopen("/usr/lib/libobjc.A.dylib",
// // 		purego.RTLD_LAZY|purego.RTLD_GLOBAL,
// // 	)
// // 	if err != nil {
// // 		panic(err)
// // 	}
// // 	// 2) Bind the two C functions
// // 	purego.RegisterLibFunc(&class_getInstanceVariable, libobjc,
// // 		"class_getInstanceVariable") // gets you an Ivar by name  [oai_citation:8‡Apple Developer](https://developer.apple.com/documentation/objectivec/1418643-class_getinstancevariable?utm_source=chatgpt.com)
// // 	purego.RegisterLibFunc(&ivar_getOffset, libobjc,
// // 		"ivar_getOffset") // gets you the byte offset  [oai_citation:9‡Apple Developer](https://developer.apple.com/documentation/objectivec/1418976-ivar_getoffset?utm_source=chatgpt.com)
// // }

// // func (d *VirtioTraditionalMemoryBalloonDevice) GetTargetVirtualMachineMemorySize() (uint64, error) {
// // 	if d.id == 0 {
// // 		return 0, fmt.Errorf("nil device")
// // 	}
// // 	// 3) Look up the ivar metadata
// // 	iv := class_getInstanceVariable(d.id.Class(), "targetVirtualMachineMemorySize")
// // 	if iv == 0 {
// // 		return 0, fmt.Errorf("ivar not found")
// // 	}
// // 	// 4) Compute its offset
// // 	offset := ivar_getOffset(iv)
// // 	// 5) Safely do pointer arithmetic
// // 	base := uintptr(d.id)
// // 	fieldPtr := objc.MethodDef
// // 	// 6) Read the raw uint64
// // 	return *(*uint64)(fieldPtr), nil
// // }
