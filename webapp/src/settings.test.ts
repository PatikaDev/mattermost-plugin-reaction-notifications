// Copyright (c) 2026-present Patika. All Rights Reserved.
// See LICENSE for license information.

import {userSettings} from './settings';

test('the setting name and values match what the server reads', () => {
    const config = userSettings('/mm');
    const setting = (config.sections[0] as any).settings[0];
    expect(config.id).toBe('dev.patika.reaction-notifications');
    expect(config.icon).toBe('/mm/plugins/dev.patika.reaction-notifications/public/icon.svg');
    expect(setting.name).toBe('notify');
    expect(setting.default).toBe('all');
    expect(setting.options.map((o: {value: string}) => o.value)).toEqual(['all', 'dms', 'off']);
});
