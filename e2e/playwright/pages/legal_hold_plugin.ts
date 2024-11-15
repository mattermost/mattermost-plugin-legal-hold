import {type Locator, type Page} from '@playwright/test';

export class LegalHoldPluginPage {
    readonly page: Page;

    readonly legalHoldPlugin: Locator;

    readonly createNewButton: Locator;
    readonly createModal: Locator;

    readonly nameField: Locator;
    readonly legalHoldName: Locator;
    readonly usernameField: Locator;
    readonly usernameDropdown: Locator;
    readonly startDate: Locator;

    readonly legalHoldButton: Locator;

    readonly verifyName: Locator;
    readonly verifyUsers: Locator;
    readonly verifyStartDate: Locator;
    readonly verifyEndDate: Locator;

    readonly verifyHoldOnPage: Locator;
    readonly releaseButton: Locator;
    readonly modalReleaseButton: Locator;

    readonly updateHold: Locator;

    constructor(page: Page) {
        this.page = page;

        // legal hold option on system console
        this.legalHoldPlugin = page.getByRole('link', {name: 'Legal Hold Plugin'});

        // create new button
        this.createNewButton = page.getByText('create new').first();

        // legal hold modal fields
        this.createModal = page.getByText('Create a new legal hold');
        this.nameField = page.getByPlaceholder('Name');
        this.legalHoldName = page.getByPlaceholder('New Legal Hold...');
        this.usernameField = page.locator('.css-19bb58m input:first-of-type');
        this.usernameDropdown = page.locator('#react-select-2-input');
        this.startDate = page.getByPlaceholder('Starting from');

        // create button
        this.legalHoldButton = page.getByRole('button', {name: 'Create legal hold'});

        // check plugin name is present
        this.verifyName = page.getByText('Sample Legal Hold').first();
        this.verifyUsers = page.getByText('1 users').first();
        this.verifyStartDate = page.getByText('Date').first();
        this.verifyEndDate = page.getByText('Never').first();

        // click release button on page
        this.verifyHoldOnPage = page.getByText('New Hold').first();
        this.releaseButton = page.getByRole('link', {name: 'Release'});

        // confirm release on modal
        this.modalReleaseButton = page.getByRole('button', {name: 'Release'});

        //update hold icon
        this.updateHold = page.locator('div:nth-child(10) > a').first();
    }

    async enterLegalHoldName(name: string) {
        await this.legalHoldName.fill(name);
    }

    async selectUsername(username: string) {
        await this.usernameDropdown.fill(username);
        await this.page.getByRole('option', {name: username}).click();
    }

    releaseHold(legalHoldName: string): Locator {
        return this.page.locator(`div:has-text("${legalHoldName}")`).getByRole('link', {name: 'Release'}).first();
    }
}

export default LegalHoldPluginPage;
