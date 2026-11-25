import {render, screen, waitFor} from '@testing-library/react';
import userEvent from '@testing-library/user-event';

import LegalHoldsSetting from './legal_holds_setting';

import Client from '@/client';
import type {LegalHold} from '@/types';

jest.mock('@/client', () => ({
    __esModule: true,
    default: {
        getLegalHolds: jest.fn(),
        releaseLegalHold: jest.fn(),
        downloadUrl: (id: string) => `/download/${id}`,
    },
}));

// The table loads profiles and groups from the Mattermost store, which these tests do not have.
jest.mock('@/components/legal_hold_table', () => ({
    __esModule: true,
    default: ({legalHolds, releaseLegalHold}: {legalHolds: LegalHold[]; releaseLegalHold: (lh: LegalHold) => void}) => (
        <ul>
            {legalHolds.map((lh) => (
                <li key={lh.id}>
                    {lh.display_name}
                    <button onClick={() => releaseLegalHold(lh)}>{`release ${lh.display_name}`}</button>
                </li>
            ))}
        </ul>
    ),
}));
jest.mock('@/components/create_legal_hold_form', () => ({
    __esModule: true,
    default: ({visible}: {visible: boolean}) => (visible ? <div>{'create form open'}</div> : null),
}));
jest.mock('@/components/update_legal_hold_form', () => ({
    __esModule: true,
    default: () => null,
}));

const legalHold = {id: 'lh1', display_name: 'Audit'} as LegalHold;

describe('LegalHoldsSetting', () => {
    beforeEach(() => {
        jest.clearAllMocks();
    });

    test('shows an empty state with a create button', async () => {
        (Client.getLegalHolds as jest.Mock).mockResolvedValue([]);
        render(<LegalHoldsSetting/>);

        expect(await screen.findByText('No legal holds')).toBeInTheDocument();

        await userEvent.click(screen.getByTestId('createNewLegalHoldOnList'));

        expect(screen.getByText('create form open')).toBeInTheDocument();
    });

    test('lists the legal holds from the server', async () => {
        (Client.getLegalHolds as jest.Mock).mockResolvedValue([legalHold]);
        render(<LegalHoldsSetting/>);

        expect(await screen.findByText('Audit')).toBeInTheDocument();
        expect(screen.queryByText('No legal holds')).not.toBeInTheDocument();
    });

    test('fetches again when the refresh button is pressed', async () => {
        (Client.getLegalHolds as jest.Mock).mockResolvedValue([legalHold]);
        render(<LegalHoldsSetting/>);
        await screen.findByText('Audit');
        expect(Client.getLegalHolds).toHaveBeenCalledTimes(1);

        await userEvent.click(screen.getByRole('button', {name: 'Refresh legal holds'}));

        await screen.findByText('Audit');
        expect(Client.getLegalHolds).toHaveBeenCalledTimes(2);
    });

    test('fetches again when a refresh is requested during a fetch', async () => {
        const stale = {...legalHold, display_name: 'Old name'};
        let finishFirst: (value: LegalHold[]) => void = () => {};
        (Client.getLegalHolds as jest.Mock).
            mockReturnValueOnce(new Promise((resolve) => {
                finishFirst = resolve;
            })).
            mockResolvedValue([legalHold]);
        render(<LegalHoldsSetting/>);
        await waitFor(() => expect(Client.getLegalHolds).toHaveBeenCalledTimes(1));

        await userEvent.click(screen.getByRole('button', {name: 'Refresh legal holds'}));
        expect(Client.getLegalHolds).toHaveBeenCalledTimes(1);
        finishFirst([stale]);

        expect(await screen.findByText('Audit')).toBeInTheDocument();
        expect(Client.getLegalHolds).toHaveBeenCalledTimes(2);
    });

    test('asks for confirmation before it releases a legal hold', async () => {
        (Client.getLegalHolds as jest.Mock).mockResolvedValue([legalHold]);
        (Client.releaseLegalHold as jest.Mock).mockResolvedValue({});
        render(<LegalHoldsSetting/>);

        await userEvent.click(await screen.findByRole('button', {name: 'release Audit'}));
        expect(Client.releaseLegalHold).not.toHaveBeenCalled();
        expect(screen.getByRole('dialog', {name: 'Release legal hold'})).toBeInTheDocument();

        await userEvent.click(screen.getByRole('button', {name: 'Release'}));

        expect(Client.releaseLegalHold).toHaveBeenCalledWith('lh1');
    });
});
