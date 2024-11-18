import {test, expect} from '@e2e-support/test_fixture';
import LegalHoldPluginPage from '../pages/legal_hold_plugin';

let user: {username: string};
let legalHoldPluginPage: LegalHoldPluginPage;

test.beforeEach(async ({pw, pages}) => {
    // login as admin user
    const {adminUser, adminClient, user: setUp} = await pw.initSetup();
    user = setUp;
    const {page} = await pw.testBrowser.login(adminUser);

    // check that plugin is enabled
    await adminClient.enablePlugin('com.mattermost.plugin-legal-hold');

    // navigate to system console
    const systemConsolePage = new pages.SystemConsolePage(page);
    await systemConsolePage.goto();
    await systemConsolePage.toBeVisible();

    // instantiate page model objects
    legalHoldPluginPage = new LegalHoldPluginPage(page);

    // scroll to legal hold page from system console
    await legalHoldPluginPage.legalHoldPlugin.click();
});

test.describe('Verify release hold', () => {
    test('Admin creates new legal hold', async () => {
        //set date to current
        const today = new Date().toISOString().split('T')[0];

        // click create new button and modal is displayed
        await legalHoldPluginPage.createNewButton.click();
        await expect(legalHoldPluginPage.createModal).toBeVisible();

        // fill in  details
        await legalHoldPluginPage.nameField.click();
        await legalHoldPluginPage.enterLegalHoldName('New Hold');
        await legalHoldPluginPage.usernameField.click();
        await legalHoldPluginPage.selectUsername(user.username);
        await legalHoldPluginPage.startDate.fill(today);

        // submit and check that modal is not visible
        await legalHoldPluginPage.legalHoldButton.click();
        await expect(legalHoldPluginPage.createModal).not.toBeVisible();

        // verify created plugin name, start and end date and user
        await expect(legalHoldPluginPage.verifyEndDate).toBeVisible();
        await expect(legalHoldPluginPage.verifyStartDate).toBeVisible();
        await expect(legalHoldPluginPage.verifyUsers).toBeVisible();
    });

    test('Admin can remove new legal hold successfully', async () => {
        // verify the created legal hold is present
        await expect(legalHoldPluginPage.verifyHoldOnPage).toHaveText('New Hold');

        // click release button on new LegalHold
        const releaseNewHold = legalHoldPluginPage.releaseHold('New Hold');
        await releaseNewHold.click();

        await legalHoldPluginPage.modalReleaseButton.click();

        // verify plugin is released
        await expect(legalHoldPluginPage.releaseHold('New Hold')).toHaveCount(0);
    });
});
