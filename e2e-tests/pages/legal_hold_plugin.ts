// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

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
        this.releaseModal = new ReleaseModal(page.getByRole('dialog', {name: 'Release legal hold'}));
    }

    // The options list is portaled to the body, so it is found from the page and not from the modal.
    async selectUsername(username: string) {
        await this.createModal.usernameInput.fill(username);
        await this.page.getByRole('option', {name: username}).click();
    }

    // Opens the calendar of a date field and clicks the day. The calendar opens on the current
    // month and labels each day with its ISO date (yyyy-mm-dd), so move to the right month first.
    async selectDate(field: Locator, isoDate: string) {
        await field.click();
        const calendar = this.page.getByRole('dialog', {name: 'Date picker'});

        const [year, month] = isoDate.split('-').map(Number);
        const today = new Date();
        const monthsAhead = (year - today.getFullYear()) * 12 + (month - 1 - today.getMonth());
        const step = calendar.getByRole('button', {name: monthsAhead > 0 ? 'Next month' : 'Previous month'});
        for (let i = 1; i <= Math.abs(monthsAhead); i++) {
            // Both months are in the DOM while the calendar animates, so wait for the new month label before the next click.
            const shown = new Date(today.getFullYear(), today.getMonth() + Math.sign(monthsAhead) * i, 1);
            const label = shown.toLocaleString('en-US', {month: 'short', year: 'numeric'});
            await step.last().click();
            await expect(calendar.getByText(label).first()).toBeVisible();
        }

        await calendar.getByRole('button', {name: isoDate, exact: true}).click();
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
    readonly usernameInput: Locator;
    readonly includePublicChannelsCheckbox: Locator;
    readonly startingFromField: Locator;
    readonly endingAtField: Locator;
    readonly clearEndDateButton: Locator;
    readonly cancelButton: Locator;
    readonly createButton: Locator;

    constructor(container: Locator) {
        this.container = container;

        this.nameInput = container.getByRole('textbox', {name: 'Name'});
        this.usernameInput = container.getByRole('combobox', {name: 'Users'});
        this.includePublicChannelsCheckbox = container.getByRole('checkbox', {name: 'Include public channels'});
        this.startingFromField = container.getByRole('combobox', {name: /Starting from/});
        this.endingAtField = container.getByRole('combobox', {name: /Ending at/});
        this.clearEndDateButton = container.getByRole('button', {name: 'Clear end date'});
        this.cancelButton = container.getByRole('button', {name: 'Cancel', exact: true});
        this.createButton = container.getByRole('button', {name: 'Create legal hold', exact: true});
    }

    async toBeVisible() {
        await expect(this.container).toBeVisible();
    }
}

class ReleaseModal {
    readonly container: Locator;

    readonly cancelButton: Locator;
    readonly releaseButton: Locator;

    constructor(container: Locator) {
        this.container = container;

        this.cancelButton = container.getByRole('button', {name: 'Cancel'});
        this.releaseButton = container.getByRole('button', {name: 'Release', exact: true});
    }

    async toBeVisible() {
        await expect(this.container).toBeVisible();
    }
}

export default LegalHoldPluginPage;
