// Copyright (c) 2026-present Patika. All Rights Reserved.
// See LICENSE for license information.

import manifest from 'manifest';

import type {PluginRegistry} from 'types/mattermost-webapp';

import {userSettings} from './settings';

declare global {
    interface Window {
        basename?: string;
        registerPlugin(pluginId: string, plugin: Plugin): void;
    }
}

export default class Plugin {
    public async initialize(registry: PluginRegistry) {
        // The notifications themselves are sent by the server; the web app only adds the
        // on/off choice to Settings.
        registry.registerUserSettings(userSettings(window.basename || ''));
    }
}

window.registerPlugin(manifest.id, new Plugin());
