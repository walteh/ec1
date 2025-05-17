#import <Foundation/Foundation.h>
#import <objc/message.h>
#import <objc/runtime.h>
#import <stdint.h>
#import <Hypervisor/Hypervisor.h>

#ifdef __cplusplus
extern "C" {
#endif

// Swift type definitions
// hv_vcpu_exit_exception_t represents a Swift struct
typedef struct {
    void* _internal;
} hv_vcpu_exit_exception_t;

// hv_boot_state represents a Swift enum
typedef enum {
    hv_boot_state_Unknown = 0,
} hv_boot_state;

// hv_reg_t represents a Swift struct
typedef struct {
    void* _internal;
} hv_reg_t;

// hv_vcpu_exit_t represents a Swift struct
typedef struct {
    void* _internal;
} hv_vcpu_exit_t;

// hv_ioapic_state represents a Swift struct
typedef struct {
    void* _internal;
} hv_ioapic_state;

// hv_sme_z_reg_t represents a Swift struct
typedef struct {
    void* _internal;
} hv_sme_z_reg_t;

// hv_apic_state_ext_t represents a Swift struct
typedef struct {
    void* _internal;
} hv_apic_state_ext_t;

// hv_atpic_state represents a Swift struct
typedef struct {
    void* _internal;
} hv_atpic_state;

// hv_feature_reg_t represents a Swift struct
typedef struct {
    void* _internal;
} hv_feature_reg_t;

// hv_gic_icv_reg_t represents a Swift struct
typedef struct {
    void* _internal;
} hv_gic_icv_reg_t;

// hv_cache_type_t represents a Swift struct
typedef struct {
    void* _internal;
} hv_cache_type_t;

// hv_gic_msi_reg_t represents a Swift struct
typedef struct {
    void* _internal;
} hv_gic_msi_reg_t;

// hv_atpic_state_ext_t represents a Swift struct
typedef struct {
    void* _internal;
} hv_atpic_state_ext_t;

// hv_interrupt_type_t represents a Swift struct
typedef struct {
    void* _internal;
} hv_interrupt_type_t;

// hv_apic_state represents a Swift struct
typedef struct {
    void* _internal;
} hv_apic_state;

// hv_vcpu_sme_state_t represents a Swift struct
typedef struct {
    void* _internal;
} hv_vcpu_sme_state_t;

// hv_sme_p_reg_t represents a Swift struct
typedef struct {
    void* _internal;
} hv_sme_p_reg_t;

// hv_sys_reg_t represents a Swift struct
typedef struct {
    void* _internal;
} hv_sys_reg_t;

// hv_gic_redistributor_reg_t represents a Swift struct
typedef struct {
    void* _internal;
} hv_gic_redistributor_reg_t;

// hv_gic_distributor_reg_t represents a Swift struct
typedef struct {
    void* _internal;
} hv_gic_distributor_reg_t;

// hv_gic_icc_reg_t represents a Swift struct
typedef struct {
    void* _internal;
} hv_gic_icc_reg_t;

// hv_simd_fp_reg_t represents a Swift struct
typedef struct {
    void* _internal;
} hv_simd_fp_reg_t;

// OS_hv_gic_config is a Swift protocol
typedef void* OS_hv_gic_config;

// OS_hv_vcpu_config is a Swift protocol
typedef void* OS_hv_vcpu_config;

// OS_hv_vm_config is a Swift protocol
typedef void* OS_hv_vm_config;

// OS_hv_gic_state is a Swift protocol
typedef void* OS_hv_gic_state;

// hv_gic_ich_reg_t represents a Swift struct
typedef struct {
    void* _internal;
} hv_gic_ich_reg_t;

// hv_gic_intid_t represents a Swift struct
typedef struct {
    void* _internal;
} hv_gic_intid_t;

// hv_exit_reason_t represents a Swift struct
typedef struct {
    void* _internal;
} hv_exit_reason_t;

// hv_ioapic_state_ext_t represents a Swift struct
typedef struct {
    void* _internal;
} hv_ioapic_state_ext_t;


// Shim for Swift property getter: apic_controls
void* c__S_hv_apic_state_FI_apic_controls_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("apic_controls"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: apic_controls
void c__S_hv_apic_state_FI_apic_controls_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setApic_controls:"), val);
}

// Shim for Swift property getter: tsc_deadline
void* c__S_hv_apic_state_FI_tsc_deadline_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("tsc_deadline"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: tsc_deadline
void c__S_hv_apic_state_FI_tsc_deadline_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setTsc_deadline:"), val);
}

// Shim for Swift property getter: exception
void* c__SA_hv_vcpu_exit_t_FI_exception_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("exception"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: exception
void c__SA_hv_vcpu_exit_t_FI_exception_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setException:"), val);
}

// Shim for Swift property getter: apic_gpa
void* c__S_hv_apic_state_FI_apic_gpa_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("apic_gpa"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: apic_gpa
void c__S_hv_apic_state_FI_apic_gpa_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setApic_gpa:"), val);
}

// Shim for Swift property getter: tpr
void* c__S_hv_apic_state_FI_tpr_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("tpr"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: tpr
void c__S_hv_apic_state_FI_tpr_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setTpr:"), val);
}

// Shim for Swift property getter: apic_id
void* c__S_hv_apic_state_FI_apic_id_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("apic_id"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: apic_id
void c__S_hv_apic_state_FI_apic_id_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setApic_id:"), val);
}

// Shim for Swift property getter: ver
void* c__S_hv_apic_state_FI_ver_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("ver"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: ver
void c__S_hv_apic_state_FI_ver_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setVer:"), val);
}

// Shim for Swift property getter: dfr
void* c__S_hv_apic_state_FI_dfr_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("dfr"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: dfr
void c__S_hv_apic_state_FI_dfr_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setDfr:"), val);
}

// Shim for Swift property getter: apr
void* c__S_hv_apic_state_FI_apr_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("apr"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: apr
void c__S_hv_apic_state_FI_apr_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setApr:"), val);
}

// Shim for Swift property getter: svr
void* c__S_hv_apic_state_FI_svr_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("svr"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: svr
void c__S_hv_apic_state_FI_svr_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setSvr:"), val);
}

// Shim for Swift property getter: ldr
void* c__S_hv_apic_state_FI_ldr_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("ldr"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: ldr
void c__S_hv_apic_state_FI_ldr_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setLdr:"), val);
}

// Shim for Swift property getter: tmr
void* c__S_hv_apic_state_FI_tmr_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("tmr"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: tmr
void c__S_hv_apic_state_FI_tmr_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setTmr:"), val);
}

// Shim for Swift property getter: isr
void* c__S_hv_apic_state_FI_isr_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("isr"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: isr
void c__S_hv_apic_state_FI_isr_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setIsr:"), val);
}

// Shim for Swift property getter: syndrome
void* c__SA_hv_vcpu_exit_exception_t_FI_syndrome_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("syndrome"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: syndrome
void c__SA_hv_vcpu_exit_exception_t_FI_syndrome_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setSyndrome:"), val);
}

// Shim for Swift property getter: lvt
void* c__S_hv_apic_state_FI_lvt_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("lvt"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: lvt
void c__S_hv_apic_state_FI_lvt_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setLvt:"), val);
}

// Shim for Swift property getter: irr
void* c__S_hv_apic_state_FI_irr_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("irr"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: irr
void c__S_hv_apic_state_FI_irr_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setIrr:"), val);
}

// Shim for Swift property getter: esr
void* c__S_hv_apic_state_FI_esr_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("esr"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: esr
void c__S_hv_apic_state_FI_esr_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setEsr:"), val);
}

// Shim for Swift property getter: icr_timer
void* c__S_hv_apic_state_FI_icr_timer_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("icr_timer"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: icr_timer
void c__S_hv_apic_state_FI_icr_timer_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setIcr_timer:"), val);
}

// Shim for Swift property getter: dcr_timer
void* c__S_hv_apic_state_FI_dcr_timer_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("dcr_timer"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: dcr_timer
void c__S_hv_apic_state_FI_dcr_timer_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setDcr_timer:"), val);
}

// Shim for Swift property getter: virtual_address
void* c__SA_hv_vcpu_exit_exception_t_FI_virtual_address_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("virtual_address"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: virtual_address
void c__SA_hv_vcpu_exit_exception_t_FI_virtual_address_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setVirtual_address:"), val);
}

// Shim for Swift property getter: icr
void* c__S_hv_apic_state_FI_icr_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("icr"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: icr
void c__S_hv_apic_state_FI_icr_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setIcr:"), val);
}

// Shim for Swift property getter: physical_address
void* c__SA_hv_vcpu_exit_exception_t_FI_physical_address_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("physical_address"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: physical_address
void c__SA_hv_vcpu_exit_exception_t_FI_physical_address_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setPhysical_address:"), val);
}

// Shim for Swift property getter: ccr_timer
void* c__S_hv_apic_state_FI_ccr_timer_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("ccr_timer"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: ccr_timer
void c__S_hv_apic_state_FI_ccr_timer_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setCcr_timer:"), val);
}

// Shim for Swift property getter: esr_pending
void* c__S_hv_apic_state_FI_esr_pending_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("esr_pending"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: esr_pending
void c__S_hv_apic_state_FI_esr_pending_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setEsr_pending:"), val);
}

// Shim for Swift property getter: rawValue
void* s_So8hv_reg_ta8rawValues6UInt32Vvp_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("rawValue"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: rawValue
void s_So8hv_reg_ta8rawValues6UInt32Vvp_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setRawValue:"), val);
}

// Shim for Swift property getter: aeoi
void* c__S_hv_apic_state_FI_aeoi_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("aeoi"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: aeoi
void c__S_hv_apic_state_FI_aeoi_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setAeoi:"), val);
}

// Shim for Swift property getter: reason
void* c__SA_hv_vcpu_exit_t_FI_reason_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("reason"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: reason
void c__SA_hv_vcpu_exit_t_FI_reason_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setReason:"), val);
}

// Shim for Swift property getter: boot_state
void* c__S_hv_apic_state_FI_boot_state_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("boot_state"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: boot_state
void c__S_hv_apic_state_FI_boot_state_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setBoot_state:"), val);
}

// Shim for Swift property getter: rawValue
void* s_So14hv_sme_z_reg_ta8rawValues6UInt32Vvp_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("rawValue"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: rawValue
void s_So14hv_sme_z_reg_ta8rawValues6UInt32Vvp_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setRawValue:"), val);
}

// Shim for Swift property getter: version
void* c__SA_hv_apic_state_ext_t_FI_version_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("version"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: version
void c__SA_hv_apic_state_ext_t_FI_version_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setVersion:"), val);
}

// Shim for Swift property getter: state
void* c__SA_hv_apic_state_ext_t_FI_state_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("state"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: state
void c__SA_hv_apic_state_ext_t_FI_state_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setState:"), val);
}

// Shim for Swift property getter: rotate
void* c__S_hv_atpic_state_FI_rotate_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("rotate"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: rotate
void c__S_hv_atpic_state_FI_rotate_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setRotate:"), val);
}

// Shim for Swift property getter: irq_base
void* c__S_hv_atpic_state_FI_irq_base_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("irq_base"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: irq_base
void c__S_hv_atpic_state_FI_irq_base_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setIrq_base:"), val);
}

// Shim for Swift property getter: rd_cmd_reg
void* c__S_hv_atpic_state_FI_rd_cmd_reg_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("rd_cmd_reg"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: rd_cmd_reg
void c__S_hv_atpic_state_FI_rd_cmd_reg_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setRd_cmd_reg:"), val);
}

// Shim for Swift property getter: icw_num
void* c__S_hv_atpic_state_FI_icw_num_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("icw_num"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: icw_num
void c__S_hv_atpic_state_FI_icw_num_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setIcw_num:"), val);
}

// Shim for Swift property getter: poll
void* c__S_hv_atpic_state_FI_poll_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("poll"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: poll
void c__S_hv_atpic_state_FI_poll_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setPoll:"), val);
}

// Shim for Swift property getter: ready
void* c__S_hv_atpic_state_FI_ready_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("ready"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: ready
void c__S_hv_atpic_state_FI_ready_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setReady:"), val);
}

// Shim for Swift property getter: aeoi
void* c__S_hv_atpic_state_FI_aeoi_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("aeoi"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: aeoi
void c__S_hv_atpic_state_FI_aeoi_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setAeoi:"), val);
}

// Shim for Swift property getter: sfn
void* c__S_hv_atpic_state_FI_sfn_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("sfn"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: sfn
void c__S_hv_atpic_state_FI_sfn_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setSfn:"), val);
}

// Shim for Swift property getter: rawValue
void* s_So16hv_gic_icv_reg_ta8rawValues6UInt16Vvp_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("rawValue"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: rawValue
void s_So16hv_gic_icv_reg_ta8rawValues6UInt16Vvp_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setRawValue:"), val);
}

// Shim for Swift property getter: rawValue
void* s_So16hv_feature_reg_ta8rawValues6UInt32Vvp_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("rawValue"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: rawValue
void s_So16hv_feature_reg_ta8rawValues6UInt32Vvp_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setRawValue:"), val);
}

// Shim for Swift property getter: rawValue
void* s_So15hv_cache_type_ta8rawValues6UInt32Vvp_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("rawValue"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: rawValue
void s_So15hv_cache_type_ta8rawValues6UInt32Vvp_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setRawValue:"), val);
}

// Shim for Swift property getter: elc
void* c__S_hv_atpic_state_FI_elc_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("elc"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: elc
void c__S_hv_atpic_state_FI_elc_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setElc:"), val);
}

// Shim for Swift property getter: mask
void* c__S_hv_atpic_state_FI_mask_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("mask"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: mask
void c__S_hv_atpic_state_FI_mask_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setMask:"), val);
}

// Shim for Swift property getter: smm
void* c__S_hv_atpic_state_FI_smm_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("smm"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: smm
void c__S_hv_atpic_state_FI_smm_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setSmm:"), val);
}

// Shim for Swift property getter: request
void* c__S_hv_atpic_state_FI_request_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("request"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: request
void c__S_hv_atpic_state_FI_request_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setRequest:"), val);
}

// Shim for Swift property getter: service
void* c__S_hv_atpic_state_FI_service_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("service"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: service
void c__S_hv_atpic_state_FI_service_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setService:"), val);
}

// Shim for Swift property getter: lowprio
void* c__S_hv_atpic_state_FI_lowprio_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("lowprio"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: lowprio
void c__S_hv_atpic_state_FI_lowprio_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setLowprio:"), val);
}

// Shim for Swift property getter: intr_raised
void* c__S_hv_atpic_state_FI_intr_raised_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("intr_raised"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: intr_raised
void c__S_hv_atpic_state_FI_intr_raised_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setIntr_raised:"), val);
}

// Shim for Swift property getter: last_request
void* c__S_hv_atpic_state_FI_last_request_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("last_request"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: last_request
void c__S_hv_atpic_state_FI_last_request_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setLast_request:"), val);
}

// Shim for Swift property getter: rawValue
void* s_So16hv_gic_msi_reg_ta8rawValues6UInt16Vvp_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("rawValue"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: rawValue
void s_So16hv_gic_msi_reg_ta8rawValues6UInt16Vvp_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setRawValue:"), val);
}

// Shim for Swift property getter: state
void* c__SA_hv_atpic_state_ext_t_FI_state_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("state"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: state
void c__SA_hv_atpic_state_ext_t_FI_state_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setState:"), val);
}

// Shim for Swift property getter: version
void* c__SA_hv_atpic_state_ext_t_FI_version_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("version"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: version
void c__SA_hv_atpic_state_ext_t_FI_version_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setVersion:"), val);
}

// Shim for Swift property getter: rtbl
void* c__S_hv_ioapic_state_FI_rtbl_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("rtbl"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: rtbl
void c__S_hv_ioapic_state_FI_rtbl_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setRtbl:"), val);
}

// Shim for Swift property getter: irr
void* c__S_hv_ioapic_state_FI_irr_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("irr"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: irr
void c__S_hv_ioapic_state_FI_irr_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setIrr:"), val);
}

// Shim for Swift property getter: rawValue
void* s_So19hv_interrupt_type_ta8rawValues6UInt32Vvp_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("rawValue"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: rawValue
void s_So19hv_interrupt_type_ta8rawValues6UInt32Vvp_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setRawValue:"), val);
}

// Shim for Swift property getter: streaming_sve_mode_enabled
void* c__SA_hv_vcpu_sme_state_t_FI_streaming_sve_mode_enabled_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("streaming_sve_mode_enabled"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: streaming_sve_mode_enabled
void c__SA_hv_vcpu_sme_state_t_FI_streaming_sve_mode_enabled_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setStreaming_sve_mode_enabled:"), val);
}

// Shim for Swift property getter: za_storage_enabled
void* c__SA_hv_vcpu_sme_state_t_FI_za_storage_enabled_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("za_storage_enabled"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: za_storage_enabled
void c__SA_hv_vcpu_sme_state_t_FI_za_storage_enabled_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setZa_storage_enabled:"), val);
}

// Shim for Swift property getter: ioa_id
void* c__S_hv_ioapic_state_FI_ioa_id_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("ioa_id"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: ioa_id
void c__S_hv_ioapic_state_FI_ioa_id_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setIoa_id:"), val);
}

// Shim for Swift property getter: ioregsel
void* c__S_hv_ioapic_state_FI_ioregsel_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("ioregsel"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: ioregsel
void c__S_hv_ioapic_state_FI_ioregsel_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setIoregsel:"), val);
}

// Shim for Swift property getter: rawValue
void* s_So12hv_sys_reg_ta8rawValues6UInt16Vvp_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("rawValue"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: rawValue
void s_So12hv_sys_reg_ta8rawValues6UInt16Vvp_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setRawValue:"), val);
}

// Shim for Swift property getter: rawValue
void* s_So14hv_sme_p_reg_ta8rawValues6UInt32Vvp_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("rawValue"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: rawValue
void s_So14hv_sme_p_reg_ta8rawValues6UInt32Vvp_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setRawValue:"), val);
}

// Shim for Swift property getter: rawValue
void* s_So26hv_gic_redistributor_reg_ta8rawValues6UInt32Vvp_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("rawValue"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: rawValue
void s_So26hv_gic_redistributor_reg_ta8rawValues6UInt32Vvp_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setRawValue:"), val);
}

// Shim for Swift property getter: rawValue
void* s_So16hv_gic_icc_reg_ta8rawValues6UInt16Vvp_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("rawValue"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: rawValue
void s_So16hv_gic_icc_reg_ta8rawValues6UInt16Vvp_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setRawValue:"), val);
}

// Shim for Swift property getter: rawValue
void* s_So24hv_gic_distributor_reg_ta8rawValues6UInt16Vvp_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("rawValue"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: rawValue
void s_So24hv_gic_distributor_reg_ta8rawValues6UInt16Vvp_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setRawValue:"), val);
}

// Shim for Swift property getter: rawValue
void* s_So16hv_simd_fp_reg_ta8rawValues6UInt32Vvp_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("rawValue"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: rawValue
void s_So16hv_simd_fp_reg_ta8rawValues6UInt32Vvp_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setRawValue:"), val);
}

// Shim for Swift property getter: version
void* c__SA_hv_ioapic_state_ext_t_FI_version_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("version"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: version
void c__SA_hv_ioapic_state_ext_t_FI_version_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setVersion:"), val);
}

// Shim for Swift property getter: state
void* c__SA_hv_ioapic_state_ext_t_FI_state_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("state"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: state
void c__SA_hv_ioapic_state_ext_t_FI_state_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setState:"), val);
}

// Shim for Swift property getter: rawValue
void* s_So16hv_gic_ich_reg_ta8rawValues6UInt16Vvp_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("rawValue"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: rawValue
void s_So16hv_gic_ich_reg_ta8rawValues6UInt16Vvp_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setRawValue:"), val);
}

// Shim for Swift property getter: rawValue
void* s_So16hv_exit_reason_ta8rawValues6UInt32Vvp_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("rawValue"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: rawValue
void s_So16hv_exit_reason_ta8rawValues6UInt32Vvp_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setRawValue:"), val);
}

// Shim for Swift property getter: rawValue
void* s_So14hv_gic_intid_ta8rawValues6UInt16Vvp_get(void* self) {
    id obj = (__bridge id)self;
    typedef id (*MsgFn)(id, SEL);
    MsgFn fn = (MsgFn)objc_msgSend;
    id rv = fn(obj, sel_getUid("rawValue"));
    return (__bridge_retained void*)rv;
}

// Shim for Swift property setter: rawValue
void s_So14hv_gic_intid_ta8rawValues6UInt16Vvp_set(void* self, void* value) {
    id obj = (__bridge id)self;
    id val = (__bridge id)value;
    typedef void (*MsgFn)(id, SEL, id);
    MsgFn fn = (MsgFn)objc_msgSend;
    fn(obj, sel_getUid("setRawValue:"), val);
}


#ifdef __cplusplus
}
#endif
