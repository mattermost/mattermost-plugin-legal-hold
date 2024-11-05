import {test} from '@e2e-support/test_fixture';
import{ CreatePluginPage} from '../pages/CreatePluginPage';


let createPluginPage : CreatePluginPage;

test('create a legal hold successfully', async ({ pw, pages }) => {
  const { adminUser, adminClient, user } = await pw.initSetup();
  const { page } = await pw.testBrowser.login(adminUser);

  await adminClient.enablePlugin('com.mattermost.plugin-legal-hold');

  const systemConsolePage = new pages.SystemConsolePage(page);
  await systemConsolePage.goto();
  await systemConsolePage.toBeVisible();
  
  await createPluginPage.clickLegalHoldPlugin();
  await createPluginPage.clickCreateNewButton();
  await createPluginPage.enterLegalHoldName('Sample Legal Hold');
  await createPluginPage.selectUsername(user.username);
  await createPluginPage.enterStartDate('2024-10-28');
  await createPluginPage.clickLegalHoldButton();
  await createPluginPage.verifyModalIsNotVisible();

});
