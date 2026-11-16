import { describe, expect, it } from 'vitest';
import { formatMemoryGB, roleName } from './roles';

describe('roleName', () => {
    it.each([
        [0, 'Guest'],
        [1, 'Member'],
        [2, 'VIP'],
        [3, 'External Member'],
        [6, 'Root'],
    ])('maps role %s to %s', (role, name) => {
        expect(roleName(role)).toBe(name);
    });

    it('handles unknown or missing roles', () => {
        expect(roleName(4)).toBe('未知角色 (4)');
        expect(roleName(undefined)).toBe('—');
        expect(roleName(null)).toBe('—');
    });
});

describe('formatMemoryGB', () => {
    const gb = 1024 * 1024 * 1024;
    it('formats bytes as GB', () => {
        expect(formatMemoryGB(1 * gb)).toBe('1 GB');
        expect(formatMemoryGB(8 * gb)).toBe('8 GB');
        expect(formatMemoryGB(1.5 * gb)).toBe('1.5 GB');
        expect(formatMemoryGB(undefined)).toBe('—');
    });
});
