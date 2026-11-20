import {useCallback} from 'react';
import {useDispatch} from 'react-redux';

import {Combobox} from '@mattermost/compass-ui/components/combobox';
import type {UserProfile} from '@mattermost/types/users';

import {getProfiles, searchProfiles} from 'mattermost-redux/actions/users';

import useAsyncOptions from '@/hooks/use_async_options';

type Props = {
    label: string;
    users: UserProfile[];
    onChange: (users: UserProfile[]) => void;
};

const getValue = (user: UserProfile) => user.id;

const getLabel = (user: UserProfile) => {
    if (user.first_name && user.last_name) {
        return `@${user.username} (${user.first_name} ${user.last_name})`;
    }
    return `@${user.username}`;
};

// UsersInput searches user profiles and lets the admin pick several of them.
const UsersInput = ({label, users, onChange}: Props) => {
    const dispatch = useDispatch();

    const search = useCallback(async (term: string) => {
        const options = {allow_inactive: true};
        const result = await dispatch(term ? searchProfiles(term, options) : getProfiles(0, 20, options));
        if (result.error) {
            throw result.error;
        }
        return result.data ?? [];
    }, [dispatch]);

    const {options, selectedOptions, selectedValues, loading, error, onInputChange, itemsForValues} = useAsyncOptions({
        search,
        getValue,
        getLabel,
        selected: users,
        errorMessage: 'Could not search users.',
    });

    return (
        <Combobox
            id='legal-hold-users'
            label={label}
            multiple={true}
            filter={false}
            loading={loading}
            options={options}
            value={selectedValues}
            selectedOptions={selectedOptions}
            emptyMessage='Type to search for users'
            message={error || undefined}
            onInputChange={onInputChange}
            onChange={(value) => onChange(itemsForValues(Array.isArray(value) ? value : []))}
        />
    );
};

export default UsersInput;
