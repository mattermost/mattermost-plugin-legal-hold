const { test, expect } = require('@playwright/test');

// Check access to plugin
test('Plugin loads and behaves correctly', async ({ page }) => {

  await page.goto('http://localhost:8065/');  


  const pluginSelector = '#plugin-container';  
  await expect(page.locator(pluginSelector)).toBeVisible();

  // Click  the plugin
  await page.click(`${pluginSelector} .plugin-button`);  


  // Check that the plugin behaves as expected
  await expect(page.locator('.plugin-output')).toHaveText('Expected Output');  

});