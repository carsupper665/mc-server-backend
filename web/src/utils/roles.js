// 角色代碼與名稱對照（對應後端 common/constants.go）
export const ROLE_ROOT = 6;

export const ROLE_NAMES = {
    0: 'Guest',
    1: 'Member',
    2: 'VIP',
    3: 'External Member',
    [ROLE_ROOT]: 'Root',
};

export const roleName = (role) => ROLE_NAMES[role] ?? (role == null ? '—' : `未知角色 (${role})`);

export const UNLIMITED_LABEL = '不限';

const GB = 1024 * 1024 * 1024;

export const formatMemoryGB = (bytes) => {
    const value = Number(bytes);
    if (!Number.isFinite(value)) return '—';
    return `${Number((value / GB).toFixed(2))} GB`;
};

// 可以由 Root 指派的角色（External Member 只能由 FGF 同步產生）
export const ASSIGNABLE_ROLES = [0, 1, 2, ROLE_ROOT];

export const roleOptions = () => ASSIGNABLE_ROLES.map(role => ({ label: ROLE_NAMES[role], value: role }));
