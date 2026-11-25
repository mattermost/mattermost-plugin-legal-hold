// Compass tokens are declared in cascade layers, so this is safe to load next to the host's copy.
// Never import '@mattermost/compass-ui/styles/standalone' here: it overrides the user's theme.
import '@mattermost/compass-ui/styles';

import type {Store} from 'redux';

import type {GlobalState} from '@mattermost/types/store';

import AmazonS3BucketSettings from '@/components/amazon_s3_bucket_settings';
import LegalHoldsSetting from '@/components/legal_holds_setting';
import {manifest} from '@/manifest';
import type {PluginRegistry} from '@/types/mattermost-webapp';

export default class Plugin {
    // eslint-disable-next-line @typescript-eslint/no-unused-vars, @typescript-eslint/no-empty-function
    public async initialize(registry: PluginRegistry, store: Store<GlobalState>) {
        // @see https://developers.mattermost.com/extend/plugins/webapp/reference/
        registry.registerAdminConsoleCustomSetting('LegalHoldsSettings', LegalHoldsSetting, {showTitle: false});
        registry.registerAdminConsoleCustomSetting('AmazonS3BucketSettings', AmazonS3BucketSettings, {showTitle: false});
    }
}

declare global {
    interface Window {
        registerPlugin(pluginId: string, plugin: Plugin): void;
    }
}

window.registerPlugin(manifest.id, new Plugin());
