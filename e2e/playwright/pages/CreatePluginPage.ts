import { type Locator, type Page, expect } from 'playwright/test';



export class CreatePluginPage{

  readonly page: Page;
  readonly legalHoldPlugin: Locator;
  readonly createNewButton: Locator;
  readonly legalHoldName: Locator;
  readonly username: Locator;
  readonly usernameDropdown: Locator;
  readonly startDate: Locator;
  readonly legalHoldButton : Locator;
  readonly modalnotVisible : Locator;
  

  constructor(page: Page){
    this.page = page;
    this.legalHoldPlugin = page.getByRole('link', { name: 'Legal Hold Plugin' });
    this.createNewButton = page.getByText('create new').first();
    this.legalHoldName = page.getByPlaceholder('New Legal Hold...');
    this.username = page.locator('.css-19bb58m')
    this.usernameDropdown = page.locator('#react-select-2-input');
    this.startDate = page.getByPlaceholder('Starting from');
    this.legalHoldButton = page.getByRole('button', { name: 'Create legal hold' })
    this.modalnotVisible = page.getByText ('Create a new legal hold')
  }

  async clickLegalHoldPlugin(){
    await this.legalHoldPlugin.click();
  }

  async clickCreateNewButton(){
    await this.createNewButton.click(); 
    await expect(this.page.getByText('Create a new legal hold')).toBeVisible;

  }
  async enterLegalHoldName (name : string){
    await this.legalHoldName.click();
    await this.legalHoldName.fill(name);
  }

  async selectUsername (username: string){
    await this.username.click()
    await this.usernameDropdown.fill(username.charAt(0));
    await this.page.getByRole('option', { name: username }).click();
  }

  
  async enterStartDate(date: string) {
    await this.startDate.fill(date);
  } 

  async clickLegalHoldButton() {
    await this.legalHoldButton.click();
  }

  async verifyModalIsNotVisible() {
    await expect(this.modalnotVisible).toBeDisabled();
  }

}
export default CreatePluginPage;

