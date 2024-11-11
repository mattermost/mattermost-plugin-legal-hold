import {test, expect} from '@e2e-support/test_fixture';
import LegalHoldPluginPage from '../pages/legal_hold_plugin';

test('Admin can remove legal hold successfully', async ({pw, pages}) => {
    // login as admin user
    const {adminUser, adminClient} = await pw.initSetup();
    const {page} = await pw.testBrowser.login(adminUser);

    // check that plugin is enabled
    await adminClient.enablePlugin('com.mattermost.plugin-legal-hold');

    // navigate to system console
    const systemConsolePage = new pages.SystemConsolePage(page);
    await systemConsolePage.goto();
    await systemConsolePage.toBeVisible();

    // instantiate page model objects
    const legalHoldPluginPage = new LegalHoldPluginPage(page);

    //on legal hold page
    await legalHoldPluginPage.legalHoldPlugin.click();
    await expect(legalHoldPluginPage.verifyHoldOnPage).toBeVisible();
    await legalHoldPluginPage.releasebutton.click();
    await legalHoldPluginPage.modalreleaseButton.click();
    await expect(legalHoldPluginPage.verifyRelease).toHaveCount(0);
});
