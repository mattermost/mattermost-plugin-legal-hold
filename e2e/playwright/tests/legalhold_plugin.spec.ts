import { test, expect, type Page } from '@playwright/test';
import fs from 'fs';

const URL = 'http://localhost:8065';
const username = 'yuney@worx4you.com';
const password = 'qwerty12345';
const workspaceName = /.*test-hub*/;

async function loginAsAdmin(page:Page) {
  await page.goto(URL);
  await page.getByRole('link', { name: 'View in Browser' }).click();

  await page.getByLabel('Email or Username').fill(username);
  await page.getByLabel('Password', { exact: true }).fill(password);
  await page.getByTestId('saveSetting').click();
  await expect(page).toHaveURL(workspaceName);
}

test.beforeEach(async ({ page }) => {
  loginAsAdmin(page);

  // Navigate to the legal hold plugin settings page
  await page.getByLabel('Product switch menu').click();
  await page.getByLabel('switcherOpen').getByRole('link', { name: 'System Console' }).click();

  // Verify that the plugin is enabled and visible
  await expect(page.getByText('Enterprise plan'), 'only enterprise plan users can use plugins').toBeVisible();
  await expect(page.locator('text=Legal Hold Plugin')).toBeVisible();
  await page.getByRole('link', { name: 'Legal Hold Plugin' }).click();
});

test.describe('Legal Hold Plugin - happy user path', () => {
  test('Admin user is able to access the plugin successfully', async ({ page }) => {
    await page.getByTestId('PluginSettings.PluginStates.com+mattermost+plugin-legal-hold.Enablefalse').check();
    await page.getByTestId('PluginSettings.PluginStates.com+mattermost+plugin-legal-hold.Enabletrue').check();
    await page.getByTestId('saveSetting').click();
    
    // Verify that the plugin is active and ready to use
    await expect(page.getByTestId('create')).toHaveCount(2);
  });

  // TODO: inserting user names result in timeout
  test('Admin user can create a new legal hold', async ({ page }) => {
    await page.getByTestId('create').click();
    await page.getByLabel('Name').fill('Testing Legal Hold');
    await page.locator('#react-select-2-input').fill('@testuser');
    await page.locator('#react-select-2-input').press('Enter');
    await page.locator('#react-select-2-input').fill('@system-bot');
    await page.locator('#react-select-2-input').press('Enter');
    await page.getByPlaceholder('Starting from').fill('2024-08-01');
    await page.getByRole('button', { name: 'Create legal hold' }).click();

    // Verify that the legal hold is created
    await expect(page.getByText('Testing Legal Hold')).toBeVisible();
  });

  test.only('Admin user can download a legal hold', async ({ page }) => {
    // Download the file
    const downloadPromise = page.waitForEvent('download');
    await page.locator('div:nth-child(10) > a:nth-child(2)').click();
    const download = await downloadPromise;

    // Wait for the download process to complete and save the downloaded file somewhere.
    // await download.saveAs('/path/to/save/at/' + download.suggestedFilename());
    console.log("file downloaded to", await download.path());

  });
  
  test('Admin user can update a legal hold', async ({ page }) => {
    await page.locator('div:nth-child(10) > a').first().click();
    await page.getByLabel('Name').fill('Testing Legal Hold Updated');
    await page.getByRole('button', { name: 'Update legal hold' }).click();

    // Verify that the legal hold is updated
    await expect(page.getByText('Testing Legal Hold Updated')).toBeVisible();
  });
  
  test('Admin user can delete/release a legal hold', async ({ page }) => {
    await page.getByRole('link', { name: 'Release' }).first().click();
    await page.getByRole('button', { name: 'Release' }).click();

    // Verify that the legal hold is deleted
    await expect(page.getByText('Testing Legal Hold Updated')).toHaveCount(0);
  });
})
