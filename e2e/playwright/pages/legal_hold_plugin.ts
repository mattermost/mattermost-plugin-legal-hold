import {expect, type Locator, type Page} from '@playwright/test';

export class LegalHoldPluginPage {
    readonly page: Page;

    readonly createModal: CreateModal;
    readonly releaseModal: ReleaseModal;

    readonly createNewButton: Locator;

    constructor(page: Page) {
        this.page = page;

        // create new button
        this.createNewButton = page.getByTestId('createNewLegalHoldOnTop');

        // create modal
        this.createModal = new CreateModal(page.getByRole('dialog', {name: 'Create a new legal hold'}));

        // release modal
        this.releaseModal = new ReleaseModal(page.getByRole('dialog', {name: 'Release Legal Hold'}));
    }

    async selectUsername(username: string) {
        await this.createModal.usernameInput.fill(username);
        await this.page.getByRole('option', {name: username}).click();
    }

    getLegalHold(name: string): Locator {
        return this.page.getByText(name);
    }

    async getLegalHoldId(name: string) {
        const legalHold = await this.getLegalHold(name);
        return await legalHold.getAttribute('data-legalholdid');
    }

    async getStartDate(name: string) {
        const id = await this.getLegalHoldId(name);
        return this.page.getByTestId(`start-date-${id}`);
    }

    async getEndDate(name: string) {
        const id = await this.getLegalHoldId(name);
        return this.page.getByTestId(`end-date-${id}`);
    }

    async getUsers(name: string) {
        const id = await this.getLegalHoldId(name);
        return this.page.getByTestId(`users-${id}`);
    }

    getUpdateButton(name: string): Locator {
        return this.page.getByLabel(`${name} update button`);
    }

    getShowSecretButton(name: string): Locator {
        return this.page.getByLabel(`${name} show secret button`);
    }

    getDownloadButton(name: string): Locator {
        return this.page.getByLabel(`${name} download button`);
    }

    getReleaseButton(name: string): Locator {
        return this.page.getByRole('button', {name: `${name} release button`});
    }
}

class CreateModal {
    readonly container: Locator;

    readonly nameInput: Locator;
    readonly usernamePlaceholder: Locator;
    readonly usernameInput: Locator;
    readonly includePublicChannelsCheckbox: Locator;
    readonly startingFromInput: Locator;
    readonly endingAtInput: Locator;
    readonly cancelButton: Locator;
    readonly createButton: Locator;

    constructor(container: Locator) {
        this.container = container;

        this.nameInput = container.getByPlaceholder('Name');
        this.usernamePlaceholder = container.locator('.css-19bb58m input:first-of-type');
        this.usernameInput = container.locator('#react-select-2-input');
        this.includePublicChannelsCheckbox = container.getByRole('checkbox', {name: 'Include public channels'});
        this.startingFromInput = container.getByPlaceholder('Starting from');
        this.endingAtInput = container.getByPlaceholder('Ending at');
        this.cancelButton = container.getByRole('button', {name: 'Cancel'});
        this.createButton = container.getByRole('button', {name: 'Create legal hold'});
    }

    async toBeVisible() {
        await expect(this.container).toBeVisible();
    }
}

class ReleaseModal {
    readonly container: Locator;

    readonly body: Locator;
    readonly cancelButton: Locator;
    readonly releaseButton: Locator;

    constructor(container: Locator) {
        this.container = container;

        this.body = container.locator('.modal-body');
        this.cancelButton = container.getByRole('button', {name: 'Cancel'});
        this.releaseButton = container.getByRole('button', {name: 'Release'});
    }

    async toBeVisible() {
        await expect(this.container).toBeVisible();
    }
}

export default LegalHoldPluginPage;
