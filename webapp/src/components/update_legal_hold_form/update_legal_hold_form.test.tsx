import {render, screen, waitFor} from '@testing-library/react';
import userEvent from '@testing-library/user-event';

import UpdateLegalHoldForm from './update_legal_hold_form';

import type {LegalHold} from '@/types';

// The store lookup and the inputs that search the server have their own coverage.
let mockEntities: {users: Array<{id: string}>; groups: Array<{id: string}>; complete: boolean};
const mockDispatch = jest.fn();
jest.mock('@/components/legal_hold_entities', () => ({
    useLegalHoldEntities: () => mockEntities,
}));
jest.mock('react-redux', () => ({
    useDispatch: () => mockDispatch,
}));
jest.mock('mattermost-redux/actions/users', () => ({
    getMissingProfilesByIds: (ids: string[]) => ({type: 'users', ids}),
}));
jest.mock('@/actions/groups', () => ({
    getMissingGroupsByIds: (ids: string[]) => ({type: 'groups', ids}),
}));
jest.mock('@/components/users_input', () => ({
    __esModule: true,
    default: ({users}: {users: Array<{id: string}>}) => <div>{`users: ${users.map((u) => u.id).join(',')}`}</div>,
}));
jest.mock('@/components/groups_input', () => ({
    __esModule: true,
    default: ({groups}: {groups: Array<{id: string}>}) => <div>{`groups: ${groups.map((g) => g.id).join(',')}`}</div>,
}));

const legalHold = {
    id: 'lh1',
    display_name: 'Audit',
    starts_at: Date.UTC(2025, 0, 15),
    ends_at: Date.UTC(2025, 5, 20),
    user_ids: ['u1'],
    group_ids: ['g1'],
    include_public_channels: true,
} as LegalHold;

beforeEach(() => {
    mockEntities = {users: [{id: 'u1'}], groups: [{id: 'g1'}], complete: true};
    mockDispatch.mockResolvedValue({data: []});
});

const setup = (updateLegalHold = jest.fn().mockResolvedValue({})) => {
    const onClose = jest.fn();
    render(
        <UpdateLegalHoldForm
            updateLegalHold={updateLegalHold}
            onClose={onClose}
            visible={true}
            legalHold={legalHold}
        />,
    );
    return {updateLegalHold, onClose};
};

describe('UpdateLegalHoldForm', () => {
    test('fills the fields from the legal hold and locks the start date', async () => {
        setup();

        expect(await screen.findByRole('textbox', {name: 'Name'})).toHaveValue('Audit');
        expect(screen.getByText('users: u1')).toBeInTheDocument();
        expect(screen.getByText('groups: g1')).toBeInTheDocument();
        expect(screen.getByRole('checkbox', {name: 'Include public channels'})).toBeChecked();
        expect(screen.getByRole('combobox', {name: /Starting at/})).toHaveAttribute('aria-disabled', 'true');
        expect(screen.getByText('01/15/2025')).toBeInTheDocument();
        expect(screen.getByText('06/20/2025')).toBeInTheDocument();
    });

    test('submits the changed name', async () => {
        const {updateLegalHold, onClose} = setup();
        const name = await screen.findByRole('textbox', {name: 'Name'});

        await userEvent.type(name, ' 2');
        await userEvent.click(screen.getByRole('button', {name: 'Update legal hold'}));

        expect(updateLegalHold).toHaveBeenCalledWith({
            id: 'lh1',
            display_name: 'Audit 2',
            user_ids: ['u1'],
            group_ids: ['g1'],
            ends_at: Date.UTC(2025, 5, 20),
            include_public_channels: true,
        });
        expect(onClose).toHaveBeenCalled();
    });

    test('removes the end date when it is cleared', async () => {
        const {updateLegalHold} = setup();

        await userEvent.click(await screen.findByRole('button', {name: 'Clear end date'}));
        await userEvent.click(screen.getByRole('button', {name: 'Update legal hold'}));

        expect(updateLegalHold).toHaveBeenCalledWith(expect.objectContaining({ends_at: 0}));
    });

    test('shows a server error', async () => {
        setup(jest.fn().mockRejectedValue(new Error('denied')));

        await userEvent.click(await screen.findByRole('button', {name: 'Update legal hold'}));

        expect(await screen.findByRole('alert')).toHaveTextContent('denied');
    });

    test('ignores a close request while the legal hold saves', async () => {
        let finish: (value: unknown) => void = () => {};
        const {onClose} = setup(jest.fn().mockReturnValue(new Promise((resolve) => {
            finish = resolve;
        })));

        await userEvent.click(await screen.findByRole('button', {name: 'Update legal hold'}));
        await userEvent.keyboard('{Escape}');

        expect(screen.getByRole('button', {name: 'Cancel'})).toBeDisabled();
        expect(onClose).not.toHaveBeenCalled();
        expect(screen.getByRole('textbox', {name: 'Name'})).toHaveValue('Audit');

        finish({});
        await waitFor(() => expect(onClose).toHaveBeenCalledTimes(1));
    });

    test('keeps the end date on the day that was saved in a time zone behind UTC', async () => {
        expect(new Date(legalHold.ends_at).getDate()).toBe(19);
        const {updateLegalHold} = setup();

        await userEvent.click(await screen.findByRole('button', {name: 'Update legal hold'}));

        expect(updateLegalHold).toHaveBeenCalledWith(expect.objectContaining({ends_at: Date.UTC(2025, 5, 20)}));
    });

    describe('when the store does not hold all the targets', () => {
        beforeEach(() => {
            mockEntities = {users: [{id: 'u1'}], groups: [], complete: false};
        });

        test('requests the missing targets, then shows them', async () => {
            setup();

            await waitFor(() => expect(screen.getByRole('textbox', {name: 'Name'})).toHaveValue('Audit'));
            expect(mockDispatch).toHaveBeenCalledWith({type: 'users', ids: ['u1']});
            expect(mockDispatch).toHaveBeenCalledWith({type: 'groups', ids: ['g1']});
        });

        test('does not save before the targets are loaded', async () => {
            mockDispatch.mockReturnValue(new Promise(() => {}));
            const {updateLegalHold} = setup();

            expect(screen.getByRole('button', {name: 'Update legal hold'})).toBeDisabled();
            expect(screen.queryByRole('textbox', {name: 'Name'})).toHaveValue('');
            expect(updateLegalHold).not.toHaveBeenCalled();
        });

        test('blocks the save and shows an error when a target fails to load', async () => {
            mockDispatch.mockResolvedValue({error: new Error('not found')});
            setup();

            expect(await screen.findByRole('alert')).toHaveTextContent('Could not load all the targeted users and groups');
            expect(screen.getByRole('button', {name: 'Update legal hold'})).toBeDisabled();
            expect(screen.getByRole('textbox', {name: 'Name'})).toHaveValue('');
        });

        test('warns when a target no longer exists', async () => {
            setup();

            expect(await screen.findByText(/1 targeted users or groups no longer exist/)).toBeInTheDocument();
        });
    });
});
