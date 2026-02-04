/**
 * @vitest-environment jsdom
 */
import { describe, it, expect } from 'vitest';
import { sanitizeErrorMessage, extractAndSanitizeError } from './errorMapping';

describe('errorMapping', () => {
    describe('sanitizeErrorMessage', () => {
        it('should return default message for empty input', () => {
            expect(sanitizeErrorMessage('')).toBe('發生未知錯誤');
            expect(sanitizeErrorMessage(null)).toBe('發生未知錯誤');
            expect(sanitizeErrorMessage(undefined)).toBe('發生未知錯誤');
        });

        it('should map exact error messages', () => {
            expect(sanitizeErrorMessage('record not found')).toBe('找不到該項目');
            expect(sanitizeErrorMessage('unauthorized')).toBe('登入已過期，請重新登入');
            expect(sanitizeErrorMessage('server already running')).toBe('伺服器已在運行中');
        });

        it('should be case insensitive for exact matches', () => {
            expect(sanitizeErrorMessage('RECORD NOT FOUND')).toBe('找不到該項目');
            expect(sanitizeErrorMessage('Unauthorized')).toBe('登入已過期，請重新登入');
        });

        it('should handle partial matches', () => {
            expect(sanitizeErrorMessage('some record not found error')).toBe('找不到該項目');
            expect(sanitizeErrorMessage('user unauthorized access')).toBe('登入已過期，請重新登入');
        });

        it('should filter profanity and return generic message', () => {
            expect(sanitizeErrorMessage('something fuck went wrong')).toBe('發生未知的伺服器錯誤');
            expect(sanitizeErrorMessage('傻逼錯誤')).toBe('發生未知的伺服器錯誤');
        });

        it('should handle SQL/database errors', () => {
            expect(sanitizeErrorMessage('SQL syntax error near...')).toBe('資料處理發生錯誤');
            expect(sanitizeErrorMessage('database connection failed')).toBe('資料處理發生錯誤');
        });

        it('should handle technical error patterns', () => {
            expect(sanitizeErrorMessage('panic: runtime error')).toBe('伺服器發生嚴重錯誤，請聯絡管理員');
            expect(sanitizeErrorMessage('nil pointer dereference')).toBe('伺服器內部錯誤');
        });

        it('should detect technical strings and return generic message', () => {
            expect(sanitizeErrorMessage('Error at 0x7fff12345678')).toBe('伺服器發生錯誤，請稍後再試');
            expect(sanitizeErrorMessage('path\\to\\file::error')).toBe('伺服器發生錯誤，請稍後再試');
        });

        it('should pass through friendly messages unchanged', () => {
            expect(sanitizeErrorMessage('請稍後再試')).toBe('請稍後再試');
            expect(sanitizeErrorMessage('操作成功')).toBe('操作成功');
        });
    });

    describe('extractAndSanitizeError', () => {
        it('should extract error from response.data.error', () => {
            const error = {
                response: {
                    data: {
                        error: 'record not found'
                    }
                }
            };
            expect(extractAndSanitizeError(error)).toBe('找不到該項目');
        });

        it('should extract error from response.data.message', () => {
            const error = {
                response: {
                    data: {
                        message: 'unauthorized'
                    }
                }
            };
            expect(extractAndSanitizeError(error)).toBe('登入已過期，請重新登入');
        });

        it('should extract error from error.message', () => {
            const error = {
                message: 'Network Error'
            };
            expect(extractAndSanitizeError(error)).toBe('Network Error');
        });

        it('should handle string errors', () => {
            expect(extractAndSanitizeError('server not running')).toBe('伺服器目前未運行');
        });
    });
});
