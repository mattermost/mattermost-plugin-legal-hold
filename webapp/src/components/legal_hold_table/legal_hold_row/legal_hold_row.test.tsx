import {render, screen, waitFor} from '@testing-library/react';
import userEvent from '@testing-library/user-event';

import LegalHoldRow from './legal_hold_row';

import type {LegalHold} from '@/types';

const legalHold: LegalHold = {
    id: 'lh1',
    name: 'audit',
    display_name: 'Audit',
    starts_at: Date.UTC(2025, 0, 15),
    ends_at: 0,
    user_ids: ['u1', 'u2'],
    group_ids: ['g1'],
    include_public_channels: false,
    secret: 'secret',
    last_execution_ended_at: 0,
    has_messages: true,
    status: 'idle',
};

const setup = (overrides: Partial<LegalHold> = {}, runLegalHold = jest.fn().mockResolvedValue({})) => {
    const props = {
        releaseLegalHold: jest.fn(),
        showUpdateModal: jest.fn(),
        showSecretModal: jest.fn(),
        runLegalHold,
        refresh: jest.fn(),
    };
    const lh = {...legalHold, ...overrides};
    render(
        <div role='table'>
            <LegalHoldRow
                legalHold={lh}
                {...props}
            />
        </div>,
    );
    return {...props, lh};
};

describe('LegalHoldRow', () => {
    test('shows the targets and the run state', () => {
        setup();

        expect(screen.getByTestId('name-lh1')).toHaveTextContent('Audit');
        expect(screen.getByTestId('users-lh1')).toHaveTextContent('2 users');
        expect(screen.getByTestId('groups-lh1')).toHaveTextContent('1 groups');
        expect(screen.getByTestId('end-date-lh1')).toHaveTextContent('Never');
        expect(screen.getByTestId('last-run-lh1')).toHaveTextContent('Never');
    });

    test('counts no targets when the server sends null lists', () => {
        setup({user_ids: null as unknown as string[], group_ids: null as unknown as string[]});

        expect(screen.getByTestId('users-lh1')).toHaveTextContent('0 users');
        expect(screen.getByTestId('groups-lh1')).toHaveTextContent('0 groups');
    });

    test('opens the update and secret dialogs and releases', async () => {
        const props = setup();

        await userEvent.click(screen.getByRole('button', {name: 'Audit update button'}));
        await userEvent.click(screen.getByRole('button', {name: 'Audit show secret button'}));
        await userEvent.click(screen.getByRole('button', {name: 'Audit release button'}));

        expect(props.showUpdateModal).toHaveBeenCalledWith(props.lh);
        expect(props.showSecretModal).toHaveBeenCalledWith(props.lh);
        expect(props.releaseLegalHold).toHaveBeenCalledWith(props.lh);
    });

    test('runs only after the user confirms', async () => {
        const props = setup();

        await userEvent.click(screen.getByRole('button', {name: 'Audit run button'}));
        expect(props.runLegalHold).not.toHaveBeenCalled();

        await userEvent.click(screen.getByRole('button', {name: 'Run now'}));

        expect(props.runLegalHold).toHaveBeenCalledWith('lh1');
    });

    test('refreshes after the run request settles', async () => {
        let finish: (value: unknown) => void = () => {};
        const props = setup({}, jest.fn().mockReturnValue(new Promise((resolve) => {
            finish = resolve;
        })));

        await userEvent.click(screen.getByRole('button', {name: 'Audit run button'}));
        await userEvent.click(screen.getByRole('button', {name: 'Run now'}));
        expect(props.refresh).not.toHaveBeenCalled();

        finish({});
        await waitFor(() => expect(props.refresh).toHaveBeenCalledTimes(1));
    });

    test('refreshes after the run request fails', async () => {
        const props = setup({}, jest.fn().mockRejectedValue(new Error('failed')));

        await userEvent.click(screen.getByRole('button', {name: 'Audit run button'}));
        await userEvent.click(screen.getByRole('button', {name: 'Run now'}));

        await waitFor(() => expect(props.refresh).toHaveBeenCalledTimes(1));
    });

    test('shows the stored dates as UTC dates', () => {
        setup({ends_at: Date.UTC(2025, 5, 20)});

        expect(new Date(Date.UTC(2025, 0, 15)).getDate()).toBe(14);

        // The row uses the default locale, so the expected text comes from the same call.
        const utcDate = (timestamp: number) => new Date(timestamp).toLocaleDateString(undefined, {timeZone: 'UTC'});
        expect(screen.getByTestId('start-date-lh1')).toHaveTextContent(utcDate(Date.UTC(2025, 0, 15)));
        expect(screen.getByTestId('end-date-lh1')).toHaveTextContent(utcDate(Date.UTC(2025, 5, 20)));
    });

    test('shows an error dialog when the run fails', async () => {
        setup({}, jest.fn().mockRejectedValue(new Error('failed')));

        await userEvent.click(screen.getByRole('button', {name: 'Audit run button'}));
        await userEvent.click(screen.getByRole('button', {name: 'Run now'}));

        expect(await screen.findByRole('dialog', {name: 'Error running legal hold'})).toBeInTheDocument();
    });

    test('disables run, release and download while the legal hold runs', () => {
        setup({status: 'executing'});

        expect(screen.getByTestId('last-run-lh1')).toHaveTextContent('Running now...');
        expect(screen.getByRole('button', {name: 'Audit run button'})).toBeDisabled();
        expect(screen.getByRole('button', {name: 'Audit release button'})).toBeDisabled();
        expect(screen.getByTestId('download-lh1')).not.toHaveAttribute('href');
    });

    test('links the download only when there are messages', () => {
        setup();
        expect(screen.getByTestId('download-lh1')).toHaveAttribute('href', expect.stringContaining('/legalholds/lh1/download'));
    });

    test('has no download link without messages', () => {
        setup({has_messages: false});
        expect(screen.getByTestId('download-lh1')).not.toHaveAttribute('href');
    });
});
