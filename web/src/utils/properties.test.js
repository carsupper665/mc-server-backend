import { describe, expect, it } from 'vitest';
import { parseProperties, serializeProperties } from './properties';

describe('properties editing', () => {
  it('updates existing keys while preserving comments, blank lines and unknown lines', () => {
    const { props, lines } = parseProperties('# Settings\nmotd=Hello=world\n\nunknown line\npvp=true');
    expect(props.motd).toBe('Hello=world');
    expect(lines[1].index).toBe(1);
    props.pvp = 'false';
    expect(serializeProperties(lines, props)).toBe('# Settings\nmotd=Hello=world\n\nunknown line\npvp=false');
  });

  it('appends absent visual keys and preserves them through subsequent edits', () => {
    const { props, lines } = parseProperties('# Settings\npvp=true\n');
    props.gamemode = 'creative';
    props['max-players'] = '30';
    const saved = serializeProperties(lines, props);
    expect(parseProperties(saved).props).toEqual({ pvp: 'true', gamemode: 'creative', 'max-players': '30' });
    props.gamemode = 'survival';
    expect(serializeProperties(lines, props).match(/gamemode=/g)).toHaveLength(1);
    expect(parseProperties(serializeProperties(lines, props)).props.gamemode).toBe('survival');
  });

  it('writes keys into an empty file', () => {
    const { props, lines } = parseProperties('');
    props.motd = '';
    expect(parseProperties(serializeProperties(lines, props)).props).toEqual({ motd: '' });
  });
});
