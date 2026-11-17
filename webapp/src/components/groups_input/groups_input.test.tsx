import {render, screen, waitFor} from '@testing-library/react';
import userEvent from '@testing-library/user-event';

import type {Group} from '@mattermost/types/groups';

import GroupsInput from './groups_input';

import Client from '@/client';

jest.mock('@/client', () => ({
    __esModule: true,
    default: {searchGroups: jest.fn()},
}));

const engineering = {id: 'g1', name: 'eng', display_name: 'Engineering'} as Group;
const sales = {id: 'g2', name: 'sales', display_name: 'sales'} as Group;

describe('GroupsInput', () => {
    test('searches through the plugin API and returns the picked group', async () => {
        (Client.searchGroups as jest.Mock).mockResolvedValue([engineering, sales]);
        const onChange = jest.fn();
        render(
            <GroupsInput
                label='LDAP Groups'
                groups={[]}
                onChange={onChange}
            />,
        );

        await userEvent.type(screen.getByRole('combobox', {name: 'LDAP Groups'}), 'e');
        await userEvent.click(await screen.findByRole('option', {name: 'Engineering - @eng'}));

        expect(Client.searchGroups).toHaveBeenCalledWith('e');
        expect(onChange).toHaveBeenLastCalledWith([engineering]);
    });

    test('omits the group name when it equals the display name', async () => {
        (Client.searchGroups as jest.Mock).mockResolvedValue([sales]);
        render(
            <GroupsInput
                label='LDAP Groups'
                groups={[]}
                onChange={jest.fn()}
            />,
        );

        await userEvent.type(screen.getByRole('combobox', {name: 'LDAP Groups'}), 's');

        expect(await screen.findByRole('option', {name: 'sales'})).toBeInTheDocument();
    });

    test('treats a null answer as no groups', async () => {
        (Client.searchGroups as jest.Mock).mockResolvedValue(null);
        render(
            <GroupsInput
                label='LDAP Groups'
                groups={[]}
                onChange={jest.fn()}
            />,
        );

        await userEvent.type(screen.getByRole('combobox', {name: 'LDAP Groups'}), 'x');

        await waitFor(() => expect(Client.searchGroups).toHaveBeenCalledWith('x'));
        expect(await screen.findByText('Type to search for LDAP groups')).toBeInTheDocument();
        expect(screen.queryByText('Could not search groups.')).not.toBeInTheDocument();
    });

    test('shows an error when the search fails', async () => {
        (Client.searchGroups as jest.Mock).mockRejectedValue(new Error('down'));
        render(
            <GroupsInput
                label='LDAP Groups'
                groups={[]}
                onChange={jest.fn()}
            />,
        );

        await userEvent.type(screen.getByRole('combobox', {name: 'LDAP Groups'}), 'x');

        expect(await screen.findByText('Could not search groups.')).toBeInTheDocument();
    });
});
