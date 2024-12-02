import {test, expect} from '@e2e-support/test_fixture';
import {getRandomId} from '@e2e-support/util';

import LegalHoldPluginPage from '../pages/legal_hold_plugin';
import {createLegalHold} from '../support/legal_hold';

test('Admin user can create a legal hold successfully', async ({pw, pages}) => {
    // Do setup and log in as admin user
    const {adminUser, adminClient, user} = await pw.initSetup();
    const {page} = await pw.testBrowser.login(adminUser);
    const pluginPage = new LegalHoldPluginPage(page);

    // Ensure plugin is enabled
    await adminClient.enablePlugin('com.mattermost.plugin-legal-hold');

    // Navigate to system console and into the legal hold plugin
    const systemConsolePage = new pages.SystemConsolePage(page);
    await systemConsolePage.goto();
    await systemConsolePage.toBeVisible();
    await systemConsolePage.sidebar.goToItem('Legal Hold Plugin');

    // Create legal hold
    const legalHoldName = `New Hold ${getRandomId()}`;
    const today = new Date().toISOString().split('T')[0];
    await createLegalHold(pluginPage, legalHoldName, [user.username], today);

    // Verify legal hold is created and details are correct
    await expect(pluginPage.getLegalHold(legalHoldName)).toBeVisible();
    const [year, month, day] = today.split('-');
    expect(await pluginPage.getStartDate(legalHoldName)).toHaveText(`${month}/${day}/${year}`);
    expect(await pluginPage.getEndDate(legalHoldName)).toHaveText('Never');
    expect(await pluginPage.getUsers(legalHoldName)).toHaveText('1 users');
});
