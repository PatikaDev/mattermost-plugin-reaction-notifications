// Copyright (c) 2026-present Patika. All Rights Reserved.
// See LICENSE for license information.

import manifest from 'manifest';

import type {PluginConfiguration} from 'types/mattermost-webapp';

// Settings → Reaction Notifications. Mattermost saves the choice as a preference (category
// "pp_<plugin id>" cut to 32 characters, name "notify"), which the server reads before sending.
export function userSettings(basename = ''): PluginConfiguration {
    return {
        id: manifest.id,
        uiName: manifest.name,
        icon: `${basename}/plugins/${manifest.id}/public/icon.svg`,
        sections: [{
            title: 'Reaction notifications',
            settings: [{
                name: 'notify',
                type: 'radio',
                title: 'Tell me when someone reacts to my messages',
                helpText: 'The Reactions bot sends you a direct message, grouping reactions that arrive within a minute. Channels you have muted are skipped.',
                default: 'all',
                options: [
                    {value: 'all', text: 'For all my messages'},
                    {value: 'dms', text: 'Only in direct and group messages'},
                    {value: 'off', text: 'Off'},
                ],
            }],
        }],
    };
}
