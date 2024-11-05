import { type Locator, type Page, expect } from 'playwright/test';


export class CreatePluginPage {

  readonly page: Page;
  readonly goToLegalHoldPluginPage: Locator;
  readonly createButton: Locator;
  readonly legalHoldName: Locator;
  readonly username: Locator;
  readonly selectUsername: Locator;
  readonly startDate: Locator;
  readonly legalHoldButton : Locator;
  readonly verifylegalHold : Locator;
  

  constructor(page: Page){
    this.page = page;
    this.goToLegalHoldPluginPage =page.getByRole('link', { name: 'Legal Hold Plugin' });
    this.createButton = page.getByText('create new').first();
    this.legalHoldName = page.getByPlaceholder('New Legal Hold...');
    this.username = page.locator('.css-19bb58m')
    this.selectUsername = page.locator('#react-select-2-input');
    this.startDate = page.getByPlaceholder('Starting from');
    this.legalHoldButton = page.getByRole('button', { name: 'Create legal hold' })
    this.verifylegalHold = page.getByText ('Create a new legal hold')
  }

  async navigateToLegalHoldPage(){
    await this.goToLegalHoldPluginPage.click();
  }

  async createNewButton(){
    await this.createButton.click(); 
    await expect(this.page.getByText('Create a new legal hold')).toBeVisible;

  }
  async enterLegalHoldName (name : string){
    await this.legalHoldName.click();
    await this.legalHoldName.fill(name);
  }

  async selectUsernameDropdown (username: string){
    await this.username.click()
    await this.selectUsername.fill(username.charAt(0));
    await this.page.getByRole('option', { name: username }).click();
  }

  
  async enterStartDate(date: string) {
    await this.startDate.fill(date);
  } 

  async createLegalHold() {
    await this.legalHoldButton.click();
  }

  async verifyLegalHoldModalIsNotVisible() {
    await expect(this.verifylegalHold).toBeDisabled();
  }

}

