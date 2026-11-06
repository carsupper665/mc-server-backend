/**
 * 解析 server.properties 格式
 */
export const parseProperties = (raw) => {
  const lines = raw.split('\n');
  const result = {};
  const lineData = [];

  lines.forEach((line, index) => {
    const trimmed = line.trim();
    
    // 保留空行和註解
    if (!trimmed || trimmed.startsWith('#')) {
      lineData.push({ type: 'comment', value: line, index });
      return;
    }

    // 解析 key=value
    const eqIndex = line.indexOf('=');
    if (eqIndex === -1) {
      lineData.push({ type: 'comment', value: line, index });
      return;
    }

    const key = line.substring(0, eqIndex).trim();
    const value = line.substring(eqIndex + 1).trim();
    
    result[key] = value;
    lineData.push({ type: 'property', key, value, index });
  });

  return { props: result, lines: lineData };
};

/**
 * 將解析後的屬性重新序列化為字串
 */
export const serializeProperties = (lines, props) => {
  const existingKeys = new Set();
  const output = lines.map(line => {
    if (line.type === 'comment') {
      return line.value;
    }
    existingKeys.add(line.key);
    return `${line.key}=${props[line.key] ?? line.value}`;
  });
  for (const key of Object.keys(props)) {
    if (!existingKeys.has(key)) {
      output.push(`${key}=${props[key]}`);
    }
  }
  return output.join('\n');
};
