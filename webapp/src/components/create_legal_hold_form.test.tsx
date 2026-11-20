import {render, screen, waitFor, within} from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import dayjs from 'dayjs';

import CreateLegalHoldForm from './create_legal_hold_form';

// The inputs that search the server have their own tests, so the form test stands in for them.
jest.mock('@/components/users_input', () => ({
    __esModule: true,
    default: ({onChange}: {onChange: (users: Array<{id: string}>) => void}) => (
        <button
            type='button'
            onClick={() => onChange([{id: 'user1'}])}
        >
            {'pick user'}
        </button>
    ),
}));
jest.mock('@/components/groups_input', () => ({
    __esModule: true,
    default: () => null,
}));

const setup = (createLegalHold = jest.fn().mockResolvedValue({})) => {
    const onClose = jest.fn();
    render(
        <CreateLegalHoldForm
            createLegalHold={createLegalHold}
            onClose={onClose}
            visible={true}
        />,
    );
    return {createLegalHold, onClose};
};

// The calendar opens on the current month and labels each day with its ISO date.
const pickToday = async (field: RegExp) => {
    await userEvent.click(screen.getByRole('combobox', {name: field}));
    await userEvent.click(await screen.findByRole('button', {name: dayjs().format('YYYY-MM-DD')}));
};

const fillValidForm = async (name = 'My Hold') => {
    await userEvent.type(screen.getByRole('textbox', {name: 'Name'}), name);
    await userEvent.click(screen.getByRole('button', {name: 'pick user'}));
    await pickToday(/Starting from/);
};

describe('CreateLegalHoldForm', () => {
    test('keeps the create button disabled until the form is valid', async () => {
        setup();
        const create = screen.getByRole('button', {name: 'Create legal hold'});
        expect(create).toBeDisabled();

        await userEvent.type(screen.getByRole('textbox', {name: 'Name'}), 'My Hold');
        expect(create).toBeDisabled();

        await userEvent.click(screen.getByRole('button', {name: 'pick user'}));
        expect(create).toBeDisabled();

        await pickToday(/Starting from/);
        expect(create).toBeEnabled();
    });

    test('submits the legal hold with a slug and no end date', async () => {
        const {createLegalHold, onClose} = setup();
        await fillValidForm('My Hold_1!');

        await userEvent.click(screen.getByRole('button', {name: 'Create legal hold'}));

        expect(createLegalHold).toHaveBeenCalledTimes(1);
        expect(createLegalHold).toHaveBeenCalledWith(expect.objectContaining({
            name: 'my-hold-1',
            display_name: 'My Hold_1!',
            user_ids: ['user1'],
            group_ids: [],
            ends_at: 0,
            include_public_channels: false,
        }));
        expect(createLegalHold.mock.calls[0][0].starts_at).toBeGreaterThan(0);
        expect(onClose).toHaveBeenCalled();
    });

    test('shows the public channel notice only when the box is checked', async () => {
        setup();
        const notice = /only captures public channels that users are members of/;
        expect(screen.queryByText(notice)).not.toBeInTheDocument();

        await userEvent.click(screen.getByRole('checkbox', {name: 'Include public channels'}));

        expect(screen.getByText(notice)).toBeInTheDocument();
    });

    test('shows a server error and stays open', async () => {
        const {onClose} = setup(jest.fn().mockRejectedValue(new Error('name taken')));
        await fillValidForm();

        await userEvent.click(screen.getByRole('button', {name: 'Create legal hold'}));

        expect(await screen.findByRole('alert')).toHaveTextContent('name taken');
        expect(onClose).not.toHaveBeenCalled();
    });

    test('closes on cancel', async () => {
        const {createLegalHold, onClose} = setup();
        const dialog = screen.getByRole('dialog', {name: 'Create a new legal hold'});

        await userEvent.click(within(dialog).getByRole('button', {name: 'Cancel'}));

        expect(createLegalHold).not.toHaveBeenCalled();
        expect(onClose).toHaveBeenCalled();
    });

    test('lets the end date be cleared', async () => {
        setup();
        expect(screen.queryByRole('button', {name: 'Clear end date'})).not.toBeInTheDocument();

        await pickToday(/Ending at/);
        await userEvent.click(screen.getByRole('button', {name: 'Clear end date'}));

        expect(screen.queryByRole('button', {name: 'Clear end date'})).not.toBeInTheDocument();
    });

    test('ignores a close request while the legal hold saves', async () => {
        let finish: (value: unknown) => void = () => {};
        const {onClose} = setup(jest.fn().mockReturnValue(new Promise((resolve) => {
            finish = resolve;
        })));
        await fillValidForm();

        await userEvent.click(screen.getByRole('button', {name: 'Create legal hold'}));
        await userEvent.keyboard('{Escape}');

        expect(screen.getByRole('button', {name: 'Cancel'})).toBeDisabled();
        expect(onClose).not.toHaveBeenCalled();
        expect(screen.getByRole('textbox', {name: 'Name'})).toHaveValue('My Hold');

        finish({});
        await waitFor(() => expect(onClose).toHaveBeenCalledTimes(1));
    });
});
