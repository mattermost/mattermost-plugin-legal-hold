import {test, expect} from '@e2e-support/test_fixture';
import {getRandomId} from '@e2e-support/util';

import PluginPage from '../pages/legal_hold_plugin';
import {createLegalHold} from '../support/legal_hold';

let pluginPage: PluginPage;
const legalHoldName = `New Hold ${getRandomId()}`;

test.beforeEach(async ({pw, pages}) => {
    // Do setup and log in as admin user
    const {adminUser, adminClient, user} = await pw.initSetup();
    const {page} = await pw.testBrowser.login(adminUser);
    pluginPage = new PluginPage(page);

    // Ensure plugin is enabled
    await adminClient.enablePlugin('com.mattermost.plugin-legal-hold');

    // Navigate to system console and into the legal hold plugin
    const systemConsolePage = new pages.SystemConsolePage(page);
    await systemConsolePage.goto();
    await systemConsolePage.toBeVisible();
    await systemConsolePage.sidebar.goToItem('Legal Hold Plugin');

    // Create legal hold
    const today = new Date().toISOString().split('T')[0];
    await createLegalHold(pluginPage, legalHoldName, [user.username], today);
});

test('Admin can release new legal hold successfully', async () => {
    // Verify the legal hold is present
    await expect(await pluginPage.getLegalHold(legalHoldName)).toBeVisible();

    // Click release button on new legal hold
    await pluginPage.getReleaseButton(legalHoldName).click();

    // Confirm release on modal
    await pluginPage.releaseModal.releaseButton.click();

    // Wait for the modal to close
    await pluginPage.releaseModal.container.waitFor({state: 'hidden'});

    // Verify legal hold is released
    await expect(pluginPage.getLegalHold(legalHoldName)).not.toBeVisible();
});
