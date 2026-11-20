import {render, screen} from '@testing-library/react';
import userEvent from '@testing-library/user-event';

import type {UserProfile} from '@mattermost/types/users';

import UsersInput from './users_input';

const mockDispatch = jest.fn();
jest.mock('react-redux', () => ({
    useDispatch: () => mockDispatch,
}));
jest.mock('mattermost-redux/actions/users', () => ({
    getProfiles: jest.fn((...args) => ({type: 'getProfiles', args})),
    searchProfiles: jest.fn((...args) => ({type: 'searchProfiles', args})),
}));

const alice = {id: 'u1', username: 'alice', first_name: 'Alice', last_name: 'Smith'} as UserProfile;
const bob = {id: 'u2', username: 'bob', first_name: '', last_name: ''} as UserProfile;

describe('UsersInput', () => {
    beforeEach(() => {
        mockDispatch.mockReset();
    });

    test('searches as the user types and returns the picked profile', async () => {
        mockDispatch.mockResolvedValue({data: [alice, bob]});
        const onChange = jest.fn();
        render(
            <UsersInput
                label='Users'
                users={[]}
                onChange={onChange}
            />,
        );

        await userEvent.type(screen.getByRole('combobox', {name: 'Users'}), 'al');
        await userEvent.click(await screen.findByRole('option', {name: '@alice (Alice Smith)'}));

        expect(mockDispatch).toHaveBeenCalledWith({type: 'searchProfiles', args: ['al', {allow_inactive: true}]});
        expect(onChange).toHaveBeenLastCalledWith([alice]);
    });

    test('shows a chip for each selected user and hides it from the options', async () => {
        mockDispatch.mockResolvedValue({data: [alice, bob]});
        render(
            <UsersInput
                label='Users'
                users={[alice]}
                onChange={jest.fn()}
            />,
        );

        expect(screen.getByText('@alice (Alice Smith)')).toBeInTheDocument();

        await userEvent.type(screen.getByRole('combobox', {name: 'Users'}), 'b');

        expect(await screen.findByRole('option', {name: '@bob'})).toBeInTheDocument();
        expect(screen.queryByRole('option', {name: '@alice (Alice Smith)'})).not.toBeInTheDocument();
    });

    test('shows an error when the search fails', async () => {
        mockDispatch.mockResolvedValue({error: new Error('down')});
        render(
            <UsersInput
                label='Users'
                users={[]}
                onChange={jest.fn()}
            />,
        );

        await userEvent.type(screen.getByRole('combobox', {name: 'Users'}), 'a');

        expect(await screen.findByText('Could not search users.')).toBeInTheDocument();
    });
});
