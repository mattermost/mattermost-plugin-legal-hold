// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {test as setup} from '@mattermost/playwright-lib';

import {legalHoldPluginId} from '@/support/constant';

setup('ensure server has license', async ({pw}) => {
    await pw.ensureLicense();
});

setup('ensure plugin is enabled', async ({pw}) => {
    await pw.ensurePluginsLoaded([legalHoldPluginId]);
});
