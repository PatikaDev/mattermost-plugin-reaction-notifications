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
    // Mattermost can call initialize twice on the same instance when a plugin is enabled again
    // while its code is still loading (e.g. during an upgrade); registering twice would show
    // every icon and button twice until the page is reloaded.
    private initialized = false;

    public uninitialize() {
        this.initialized = false;
    }

    public async initialize(registry: PluginRegistry) {
        if (this.initialized) {
            return;
        }
        this.initialized = true;

        // The notifications themselves are sent by the server; the web app only adds the
        // on/off choice to Settings.
        registry.registerUserSettings(userSettings(window.basename || ''));
    }
}

window.registerPlugin(manifest.id, new Plugin());
