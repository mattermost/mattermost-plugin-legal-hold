
import {test, expect} from '@e2e-support/test_fixture';

test('create a legal hold successfully', async ({pw, pages}) => {

  const {adminUser, adminClient, user} = await pw.initSetup();
  const {page} = await pw.testBrowser.login(adminUser); 

  await adminClient.enablePlugin('com.mattermost.plugin-legal-hold');

   const systemConsolePage = new pages.SystemConsolePage(page);
   await systemConsolePage.goto();
   await systemConsolePage.toBeVisible();

  // navigate to plugin page
   await page.getByRole('link', { name: 'Legal Hold Plugin' }).click();
  
    //click the create new button
    await page.getByText('create new').first().click();
    await expect (page.getByText ('Create a new legal hold')).toBeVisible
    
    //enter legal hold name
    await page.getByPlaceholder('Name').click();
    await page.getByPlaceholder('New Legal Hold...').fill('Sample Legal Hold');
  
     // select user
    await page.locator('.css-19bb58m').click();
    await page.locator('#react-select-2-input').fill('s');
    await page.getByRole('option', { name: user.username }).click();
    
    //enter start date
    await page.getByPlaceholder('Starting from').fill('2024-10-28');
  
    //click the create legal hold button
    await page.getByRole('button', { name: 'Create legal hold' }).click();
  
    //verify that the create legal hold modal is no longer visible
    await expect (page.getByText ('Create a new legal hold')).toBeDisabled
  
  });

test('Check newly created plugin', async ({page}) => {

    expect(page.getByText('Sample Legal Hold')).toBeVisible;
    expect(page.getByText('Release')).toBeVisible;
});
