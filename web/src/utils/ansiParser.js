/**
 * ANSI Escape Code 解析器
 * 將後端回傳的 ANSI 字串轉換為帶有顏色的 HTML
 */

// ANSI 顏色對照表
const ANSI_COLORS = {
    // 前景色
    '30': '#000000', // Black
    '31': '#e74c3c', // Red
    '32': '#2ecc71', // Green
    '33': '#f1c40f', // Yellow
    '34': '#3498db', // Blue
    '35': '#9b59b6', // Magenta
    '36': '#1abc9c', // Cyan
    '37': '#ecf0f1', // White
    '90': '#7f8c8d', // Bright Black (Gray)
    '91': '#ff6b6b', // Bright Red
    '92': '#4ade80', // Bright Green
    '93': '#fcd34d', // Bright Yellow
    '94': '#60a5fa', // Bright Blue
    '95': '#c084fc', // Bright Magenta
    '96': '#22d3d8', // Bright Cyan
    '97': '#ffffff', // Bright White
};

// 背景色對照表
const ANSI_BG_COLORS = {
    '40': '#000000',
    '41': '#e74c3c',
    '42': '#2ecc71',
    '43': '#f1c40f',
    '44': '#3498db',
    '45': '#9b59b6',
    '46': '#1abc9c',
    '47': '#ecf0f1',
};

/**
 * 解析 ANSI 字串並轉換為 HTML
 * @param {string} text - 包含 ANSI Escape Codes 的字串
 * @returns {string} - 帶有 HTML span 標籤的字串
 */
export function parseAnsiToHtml(text) {
    if (!text) return '';

    // ANSI Escape Code 正規表達式
    // 匹配 \x1b[...m 或 \033[...m 格式
    const ansiRegex = /\x1b\[([0-9;]*)m/g;

    let result = '';
    let lastIndex = 0;
    let currentStyles = {
        color: null,
        backgroundColor: null,
        bold: false,
        italic: false,
        underline: false,
    };
    let spanOpen = false;

    // 處理每個 ANSI Escape Code
    text.replace(ansiRegex, (match, codes, offset) => {
        // 添加 ANSI Code 之前的文字
        if (offset > lastIndex) {
            const textContent = escapeHtml(text.substring(lastIndex, offset));
            if (spanOpen) {
                result += textContent;
            } else if (hasActiveStyles(currentStyles)) {
                result += `<span style="${buildStyleString(currentStyles)}">${textContent}`;
                spanOpen = true;
            } else {
                result += textContent;
            }
        }

        // 關閉之前的 span
        if (spanOpen) {
            result += '</span>';
            spanOpen = false;
        }

        // 解析並應用新的樣式
        const codeArray = codes.split(';').filter(c => c !== '');

        for (const code of codeArray) {
            switch (code) {
                case '0': // Reset
                    currentStyles = {
                        color: null,
                        backgroundColor: null,
                        bold: false,
                        italic: false,
                        underline: false,
                    };
                    break;
                case '1': // Bold
                    currentStyles.bold = true;
                    break;
                case '3': // Italic
                    currentStyles.italic = true;
                    break;
                case '4': // Underline
                    currentStyles.underline = true;
                    break;
                case '22': // Normal intensity (not bold)
                    currentStyles.bold = false;
                    break;
                case '23': // Not italic
                    currentStyles.italic = false;
                    break;
                case '24': // Not underline
                    currentStyles.underline = false;
                    break;
                default:
                    // 前景色
                    if (ANSI_COLORS[code]) {
                        currentStyles.color = ANSI_COLORS[code];
                    }
                    // 背景色
                    if (ANSI_BG_COLORS[code]) {
                        currentStyles.backgroundColor = ANSI_BG_COLORS[code];
                    }
                    break;
            }
        }

        lastIndex = offset + match.length;
        return match;
    });

    // 添加剩餘的文字
    if (lastIndex < text.length) {
        const remainingText = escapeHtml(text.substring(lastIndex));
        if (hasActiveStyles(currentStyles)) {
            result += `<span style="${buildStyleString(currentStyles)}">${remainingText}</span>`;
        } else {
            result += remainingText;
        }
    } else if (spanOpen) {
        result += '</span>';
    }

    return result;
}

/**
 * 檢查是否有活躍的樣式
 */
function hasActiveStyles(styles) {
    return styles.color || styles.backgroundColor || styles.bold || styles.italic || styles.underline;
}

/**
 * 構建 CSS 樣式字串
 */
function buildStyleString(styles) {
    const parts = [];
    if (styles.color) parts.push(`color: ${styles.color}`);
    if (styles.backgroundColor) parts.push(`background-color: ${styles.backgroundColor}`);
    if (styles.bold) parts.push('font-weight: bold');
    if (styles.italic) parts.push('font-style: italic');
    if (styles.underline) parts.push('text-decoration: underline');
    return parts.join('; ');
}

/**
 * 轉義 HTML 特殊字元
 */
function escapeHtml(text) {
    const htmlEntities = {
        '&': '&amp;',
        '<': '&lt;',
        '>': '&gt;',
        '"': '&quot;',
        "'": '&#39;',
    };
    return text.replace(/[&<>"']/g, char => htmlEntities[char]);
}

/**
 * 移除所有 ANSI Escape Codes（用於純文字顯示）
 */
export function stripAnsi(text) {
    if (!text) return '';
    return text.replace(/\x1b\[[0-9;]*m/g, '');
}
