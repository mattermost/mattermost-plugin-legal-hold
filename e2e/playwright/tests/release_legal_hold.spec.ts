import {expect, test} from 'mmtest_playwright-lib';

import PluginPage from '../pages/legal_hold_plugin';
import {createLegalHold} from '../support/legal_hold';

let pluginPage: PluginPage;
let legalHoldName: string;

test.beforeEach(async ({pw}) => {
    // Do setup and log in as admin user
    const {adminUser, adminClient, user} = await pw.initSetup();
    const {page, systemConsolePage} = await pw.testBrowser.login(adminUser);
    pluginPage = new PluginPage(page);

    // Ensure plugin is enabled
    await adminClient.enablePlugin('com.mattermost.plugin-legal-hold');

    // Navigate to system console and into the legal hold plugin
    await systemConsolePage.goto();
    await systemConsolePage.toBeVisible();
    await systemConsolePage.sidebar.goToItem('Legal Hold Plugin');

    // Create legal hold
    legalHoldName = `New Hold ${pw.random.id()}`;
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
