import {expect, test as setup} from '@mattermost/playwright-lib';
import {Client4} from '@mattermost/client';

import {legalHoldPluginId} from '@support/constant';

setup('ensure server has license', async ({pw}) => {
    const {adminClient} = await pw.getAdminClient();
    expect(await ensureLicense(adminClient)).toBe(true);
});

setup('ensure plugin is enabled', async ({pw}) => {
    const {adminClient} = await pw.getAdminClient();

    const pluginStatus = await adminClient.getPluginStatuses();
    const plugins = await adminClient.getPlugins();

    for (const pluginId of [legalHoldPluginId]) {
        const isInstalled = pluginStatus.some(({plugin_id}) => plugin_id === pluginId);

        if (!isInstalled) {
            // eslint-disable-next-line no-console
            console.log(`${pluginId} is not installed. Related visual test will fail.`);
            continue;
        }

        const isActive = plugins.active.some(({id}) => id === pluginId);

        if (!isActive) {
            await adminClient.enablePlugin(pluginId);
            // eslint-disable-next-line no-console
            console.log(`${pluginId} is installed and has been activated.`);
        } else {
            // eslint-disable-next-line no-console
            console.log(`${pluginId} is installed and active.`);
        }
    }
});

async function ensureLicense(adminClient: Client4) {
    try {
        const currentLicense = await adminClient.getClientLicenseOld();

        if (currentLicense?.IsLicensed === 'true') {
            return true;
        }

        await requestTrialLicense(adminClient);

        const trialLicense = await adminClient.getClientLicenseOld();
        return trialLicense?.IsLicensed === 'true';
    } catch (error) {
        // eslint-disable-next-line no-console
        console.error('Error ensuring license', error);
        return false;
    }
}

async function requestTrialLicense(adminClient: Client4) {
    try {
        await adminClient.requestTrialLicense({
            receive_emails_accepted: true,
            terms_accepted: true,
            users: 100,
            company_country: 'US',
            contact_email: process.env.MM_ADMIN_EMAIL ?? '',
            contact_name: 'Test Mattermost',
            company_name: 'MattermostTest',
            company_size: '1-10',
        });
    } catch (e) {
        // eslint-disable-next-line no-console
        console.error('Failed to request trial license', e);
        throw e;
    }
}
