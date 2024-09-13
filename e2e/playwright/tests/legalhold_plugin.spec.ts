import {test, expect} from '@e2e-support/test_fixture';

test('Admin user is able to access the plugin successfully', async ({pw, pages}) => {
    // # Log in as admin
    const {adminUser} = await pw.initSetup();
    const {page} = await pw.testBrowser.login(adminUser);

    // # Visit system console
    const systemConsolePage = new pages.SystemConsolePage(page);
    await systemConsolePage.goto();
    await systemConsolePage.toBeVisible();

    // # Go to PLUGINS > Legal Hold Plugin
    await systemConsolePage.page.getByRole('link', {name: 'Legal Hold Plugin'}).click();

    // # Enable the plugin
    await page.getByTestId('PluginSettings.PluginStates.com+mattermost+plugin-legal-hold.Enabletrue').check();
    await page.getByTestId('saveSetting').click();

    // * Verify that the plugin is active and ready to use
    await expect(page.getByTestId('create')).toHaveCount(2);
});