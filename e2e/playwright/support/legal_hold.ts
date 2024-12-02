import {expect} from '@e2e-support/test_fixture';

import PluginPage from '../pages/legal_hold_plugin';

export async function createLegalHold(
    pluginPage: PluginPage,
    name: string,
    usernames: string[],
    startDate: string,
    includePublicChannels = false,
    endDate = '',
) {
    // Click create new button and check that modal is displayed
    await pluginPage.createNewButton.click();
    await pluginPage.createModal.toBeVisible();

    // Enter name
    await pluginPage.createModal.nameInput.fill(name);

    // Select users
    for (const username of usernames) {
        await pluginPage.createModal.usernameInput.fill(username);
        await pluginPage.selectUsername(username);
    }

    // Enter start and end date
    await pluginPage.createModal.startingFromInput.fill(startDate);

    if (endDate) {
        await pluginPage.createModal.endingAtInput.fill(endDate);
    }

    // Set wether to include public channels
    if (includePublicChannels) {
        await pluginPage.createModal.includePublicChannelsCheckbox.check();
    } else {
        await pluginPage.createModal.includePublicChannelsCheckbox.uncheck();
    }

    // Click create and check that modal is not visible
    await expect(pluginPage.createModal.createButton).toBeEnabled();
    await pluginPage.createModal.createButton.click();
    await pluginPage.createModal.container.waitFor({state: 'hidden'});
}
