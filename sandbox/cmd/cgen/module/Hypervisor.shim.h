#ifndef HYPERVISOR_SHIM_H
#define HYPERVISOR_SHIM_H

#include <stdint.h>

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


// Swift property: apic_controls
void* c__S_hv_apic_state_FI_apic_controls_get(void* self);
void c__S_hv_apic_state_FI_apic_controls_set(void* self, void* value);

// Swift property: tsc_deadline
void* c__S_hv_apic_state_FI_tsc_deadline_get(void* self);
void c__S_hv_apic_state_FI_tsc_deadline_set(void* self, void* value);

// Swift property: exception
void* c__SA_hv_vcpu_exit_t_FI_exception_get(void* self);
void c__SA_hv_vcpu_exit_t_FI_exception_set(void* self, void* value);

// Swift property: apic_gpa
void* c__S_hv_apic_state_FI_apic_gpa_get(void* self);
void c__S_hv_apic_state_FI_apic_gpa_set(void* self, void* value);

// Swift property: tpr
void* c__S_hv_apic_state_FI_tpr_get(void* self);
void c__S_hv_apic_state_FI_tpr_set(void* self, void* value);

// Swift property: apic_id
void* c__S_hv_apic_state_FI_apic_id_get(void* self);
void c__S_hv_apic_state_FI_apic_id_set(void* self, void* value);

// Swift property: ver
void* c__S_hv_apic_state_FI_ver_get(void* self);
void c__S_hv_apic_state_FI_ver_set(void* self, void* value);

// Swift property: dfr
void* c__S_hv_apic_state_FI_dfr_get(void* self);
void c__S_hv_apic_state_FI_dfr_set(void* self, void* value);

// Swift property: apr
void* c__S_hv_apic_state_FI_apr_get(void* self);
void c__S_hv_apic_state_FI_apr_set(void* self, void* value);

// Swift property: svr
void* c__S_hv_apic_state_FI_svr_get(void* self);
void c__S_hv_apic_state_FI_svr_set(void* self, void* value);

// Swift property: ldr
void* c__S_hv_apic_state_FI_ldr_get(void* self);
void c__S_hv_apic_state_FI_ldr_set(void* self, void* value);

// Swift property: tmr
void* c__S_hv_apic_state_FI_tmr_get(void* self);
void c__S_hv_apic_state_FI_tmr_set(void* self, void* value);

// Swift property: isr
void* c__S_hv_apic_state_FI_isr_get(void* self);
void c__S_hv_apic_state_FI_isr_set(void* self, void* value);

// Swift property: syndrome
void* c__SA_hv_vcpu_exit_exception_t_FI_syndrome_get(void* self);
void c__SA_hv_vcpu_exit_exception_t_FI_syndrome_set(void* self, void* value);

// Swift property: lvt
void* c__S_hv_apic_state_FI_lvt_get(void* self);
void c__S_hv_apic_state_FI_lvt_set(void* self, void* value);

// Swift property: irr
void* c__S_hv_apic_state_FI_irr_get(void* self);
void c__S_hv_apic_state_FI_irr_set(void* self, void* value);

// Swift property: esr
void* c__S_hv_apic_state_FI_esr_get(void* self);
void c__S_hv_apic_state_FI_esr_set(void* self, void* value);

// Swift property: icr_timer
void* c__S_hv_apic_state_FI_icr_timer_get(void* self);
void c__S_hv_apic_state_FI_icr_timer_set(void* self, void* value);

// Swift property: dcr_timer
void* c__S_hv_apic_state_FI_dcr_timer_get(void* self);
void c__S_hv_apic_state_FI_dcr_timer_set(void* self, void* value);

// Swift property: virtual_address
void* c__SA_hv_vcpu_exit_exception_t_FI_virtual_address_get(void* self);
void c__SA_hv_vcpu_exit_exception_t_FI_virtual_address_set(void* self, void* value);

// Swift property: icr
void* c__S_hv_apic_state_FI_icr_get(void* self);
void c__S_hv_apic_state_FI_icr_set(void* self, void* value);

// Swift property: physical_address
void* c__SA_hv_vcpu_exit_exception_t_FI_physical_address_get(void* self);
void c__SA_hv_vcpu_exit_exception_t_FI_physical_address_set(void* self, void* value);

// Swift property: ccr_timer
void* c__S_hv_apic_state_FI_ccr_timer_get(void* self);
void c__S_hv_apic_state_FI_ccr_timer_set(void* self, void* value);

// Swift property: esr_pending
void* c__S_hv_apic_state_FI_esr_pending_get(void* self);
void c__S_hv_apic_state_FI_esr_pending_set(void* self, void* value);

// Swift property: rawValue
void* s_So8hv_reg_ta8rawValues6UInt32Vvp_get(void* self);
void s_So8hv_reg_ta8rawValues6UInt32Vvp_set(void* self, void* value);

// Swift property: aeoi
void* c__S_hv_apic_state_FI_aeoi_get(void* self);
void c__S_hv_apic_state_FI_aeoi_set(void* self, void* value);

// Swift property: reason
void* c__SA_hv_vcpu_exit_t_FI_reason_get(void* self);
void c__SA_hv_vcpu_exit_t_FI_reason_set(void* self, void* value);

// Swift property: boot_state
void* c__S_hv_apic_state_FI_boot_state_get(void* self);
void c__S_hv_apic_state_FI_boot_state_set(void* self, void* value);

// Swift property: rawValue
void* s_So14hv_sme_z_reg_ta8rawValues6UInt32Vvp_get(void* self);
void s_So14hv_sme_z_reg_ta8rawValues6UInt32Vvp_set(void* self, void* value);

// Swift property: version
void* c__SA_hv_apic_state_ext_t_FI_version_get(void* self);
void c__SA_hv_apic_state_ext_t_FI_version_set(void* self, void* value);

// Swift property: state
void* c__SA_hv_apic_state_ext_t_FI_state_get(void* self);
void c__SA_hv_apic_state_ext_t_FI_state_set(void* self, void* value);

// Swift property: rotate
void* c__S_hv_atpic_state_FI_rotate_get(void* self);
void c__S_hv_atpic_state_FI_rotate_set(void* self, void* value);

// Swift property: irq_base
void* c__S_hv_atpic_state_FI_irq_base_get(void* self);
void c__S_hv_atpic_state_FI_irq_base_set(void* self, void* value);

// Swift property: rd_cmd_reg
void* c__S_hv_atpic_state_FI_rd_cmd_reg_get(void* self);
void c__S_hv_atpic_state_FI_rd_cmd_reg_set(void* self, void* value);

// Swift property: icw_num
void* c__S_hv_atpic_state_FI_icw_num_get(void* self);
void c__S_hv_atpic_state_FI_icw_num_set(void* self, void* value);

// Swift property: poll
void* c__S_hv_atpic_state_FI_poll_get(void* self);
void c__S_hv_atpic_state_FI_poll_set(void* self, void* value);

// Swift property: ready
void* c__S_hv_atpic_state_FI_ready_get(void* self);
void c__S_hv_atpic_state_FI_ready_set(void* self, void* value);

// Swift property: aeoi
void* c__S_hv_atpic_state_FI_aeoi_get(void* self);
void c__S_hv_atpic_state_FI_aeoi_set(void* self, void* value);

// Swift property: sfn
void* c__S_hv_atpic_state_FI_sfn_get(void* self);
void c__S_hv_atpic_state_FI_sfn_set(void* self, void* value);

// Swift property: rawValue
void* s_So16hv_gic_icv_reg_ta8rawValues6UInt16Vvp_get(void* self);
void s_So16hv_gic_icv_reg_ta8rawValues6UInt16Vvp_set(void* self, void* value);

// Swift property: rawValue
void* s_So16hv_feature_reg_ta8rawValues6UInt32Vvp_get(void* self);
void s_So16hv_feature_reg_ta8rawValues6UInt32Vvp_set(void* self, void* value);

// Swift property: rawValue
void* s_So15hv_cache_type_ta8rawValues6UInt32Vvp_get(void* self);
void s_So15hv_cache_type_ta8rawValues6UInt32Vvp_set(void* self, void* value);

// Swift property: elc
void* c__S_hv_atpic_state_FI_elc_get(void* self);
void c__S_hv_atpic_state_FI_elc_set(void* self, void* value);

// Swift property: mask
void* c__S_hv_atpic_state_FI_mask_get(void* self);
void c__S_hv_atpic_state_FI_mask_set(void* self, void* value);

// Swift property: smm
void* c__S_hv_atpic_state_FI_smm_get(void* self);
void c__S_hv_atpic_state_FI_smm_set(void* self, void* value);

// Swift property: request
void* c__S_hv_atpic_state_FI_request_get(void* self);
void c__S_hv_atpic_state_FI_request_set(void* self, void* value);

// Swift property: service
void* c__S_hv_atpic_state_FI_service_get(void* self);
void c__S_hv_atpic_state_FI_service_set(void* self, void* value);

// Swift property: lowprio
void* c__S_hv_atpic_state_FI_lowprio_get(void* self);
void c__S_hv_atpic_state_FI_lowprio_set(void* self, void* value);

// Swift property: intr_raised
void* c__S_hv_atpic_state_FI_intr_raised_get(void* self);
void c__S_hv_atpic_state_FI_intr_raised_set(void* self, void* value);

// Swift property: last_request
void* c__S_hv_atpic_state_FI_last_request_get(void* self);
void c__S_hv_atpic_state_FI_last_request_set(void* self, void* value);

// Swift property: rawValue
void* s_So16hv_gic_msi_reg_ta8rawValues6UInt16Vvp_get(void* self);
void s_So16hv_gic_msi_reg_ta8rawValues6UInt16Vvp_set(void* self, void* value);

// Swift property: state
void* c__SA_hv_atpic_state_ext_t_FI_state_get(void* self);
void c__SA_hv_atpic_state_ext_t_FI_state_set(void* self, void* value);

// Swift property: version
void* c__SA_hv_atpic_state_ext_t_FI_version_get(void* self);
void c__SA_hv_atpic_state_ext_t_FI_version_set(void* self, void* value);

// Swift property: rtbl
void* c__S_hv_ioapic_state_FI_rtbl_get(void* self);
void c__S_hv_ioapic_state_FI_rtbl_set(void* self, void* value);

// Swift property: irr
void* c__S_hv_ioapic_state_FI_irr_get(void* self);
void c__S_hv_ioapic_state_FI_irr_set(void* self, void* value);

// Swift property: rawValue
void* s_So19hv_interrupt_type_ta8rawValues6UInt32Vvp_get(void* self);
void s_So19hv_interrupt_type_ta8rawValues6UInt32Vvp_set(void* self, void* value);

// Swift property: streaming_sve_mode_enabled
void* c__SA_hv_vcpu_sme_state_t_FI_streaming_sve_mode_enabled_get(void* self);
void c__SA_hv_vcpu_sme_state_t_FI_streaming_sve_mode_enabled_set(void* self, void* value);

// Swift property: za_storage_enabled
void* c__SA_hv_vcpu_sme_state_t_FI_za_storage_enabled_get(void* self);
void c__SA_hv_vcpu_sme_state_t_FI_za_storage_enabled_set(void* self, void* value);

// Swift property: ioa_id
void* c__S_hv_ioapic_state_FI_ioa_id_get(void* self);
void c__S_hv_ioapic_state_FI_ioa_id_set(void* self, void* value);

// Swift property: ioregsel
void* c__S_hv_ioapic_state_FI_ioregsel_get(void* self);
void c__S_hv_ioapic_state_FI_ioregsel_set(void* self, void* value);

// Swift property: rawValue
void* s_So12hv_sys_reg_ta8rawValues6UInt16Vvp_get(void* self);
void s_So12hv_sys_reg_ta8rawValues6UInt16Vvp_set(void* self, void* value);

// Swift property: rawValue
void* s_So14hv_sme_p_reg_ta8rawValues6UInt32Vvp_get(void* self);
void s_So14hv_sme_p_reg_ta8rawValues6UInt32Vvp_set(void* self, void* value);

// Swift property: rawValue
void* s_So26hv_gic_redistributor_reg_ta8rawValues6UInt32Vvp_get(void* self);
void s_So26hv_gic_redistributor_reg_ta8rawValues6UInt32Vvp_set(void* self, void* value);

// Swift property: rawValue
void* s_So16hv_gic_icc_reg_ta8rawValues6UInt16Vvp_get(void* self);
void s_So16hv_gic_icc_reg_ta8rawValues6UInt16Vvp_set(void* self, void* value);

// Swift property: rawValue
void* s_So24hv_gic_distributor_reg_ta8rawValues6UInt16Vvp_get(void* self);
void s_So24hv_gic_distributor_reg_ta8rawValues6UInt16Vvp_set(void* self, void* value);

// Swift property: rawValue
void* s_So16hv_simd_fp_reg_ta8rawValues6UInt32Vvp_get(void* self);
void s_So16hv_simd_fp_reg_ta8rawValues6UInt32Vvp_set(void* self, void* value);

// Swift property: version
void* c__SA_hv_ioapic_state_ext_t_FI_version_get(void* self);
void c__SA_hv_ioapic_state_ext_t_FI_version_set(void* self, void* value);

// Swift property: state
void* c__SA_hv_ioapic_state_ext_t_FI_state_get(void* self);
void c__SA_hv_ioapic_state_ext_t_FI_state_set(void* self, void* value);

// Swift property: rawValue
void* s_So16hv_gic_ich_reg_ta8rawValues6UInt16Vvp_get(void* self);
void s_So16hv_gic_ich_reg_ta8rawValues6UInt16Vvp_set(void* self, void* value);

// Swift property: rawValue
void* s_So16hv_exit_reason_ta8rawValues6UInt32Vvp_get(void* self);
void s_So16hv_exit_reason_ta8rawValues6UInt32Vvp_set(void* self, void* value);

// Swift property: rawValue
void* s_So14hv_gic_intid_ta8rawValues6UInt16Vvp_get(void* self);
void s_So14hv_gic_intid_ta8rawValues6UInt16Vvp_set(void* self, void* value);


#ifdef __cplusplus
}
#endif

#endif // HYPERVISOR_SHIM_H
