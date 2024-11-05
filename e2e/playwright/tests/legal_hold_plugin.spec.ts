import {test, expect} from '@e2e-support/test_fixture';
import LegalHoldPluginPage from '../pages/legal_hold_plugin';

let legalHoldPluginPage: LegalHoldPluginPage;

test('create a legal hold successfully', async ({pw, pages}) => {
    //login as admin user
    const {adminUser, adminClient, user} = await pw.initSetup();
    const {page} = await pw.testBrowser.login(adminUser);

    //check that plugin is enabled
    await adminClient.enablePlugin('com.mattermost.plugin-legal-hold');

    // navigate to system console
    const systemConsolePage = new pages.SystemConsolePage(page);
    await systemConsolePage.goto();
    await systemConsolePage.toBeVisible();

    // instansiate page model objects
    legalHoldPluginPage = new LegalHoldPluginPage(page);

    //scroll to legal hold page from system console
    await legalHoldPluginPage.legalHoldPlugin.click();

    //click create new button and modal is displayed
    await legalHoldPluginPage.createNewButton.click();
    await expect(legalHoldPluginPage.createModal).toBeVisible();

    // fill in  details
    await legalHoldPluginPage.nameField.click();
    await legalHoldPluginPage.enterLegalHoldName('Sample Legal Hold');
    await legalHoldPluginPage.username.click();
    await legalHoldPluginPage.selectUsername(user.username);
    await legalHoldPluginPage.enterStartDate('2024-10-28');

    //submit and check that modal is not visible
    await legalHoldPluginPage.legalHoldButton.click();
    await expect(legalHoldPluginPage.createModal).not.toBeVisible();
});
